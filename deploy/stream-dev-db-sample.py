#!/usr/bin/env python3
"""Read-only PostgreSQL development sample; run via SSH stdin on goodserver.

Streams gzip-compressed SQL to stdout without any remote staging files. All
pg_dump/COPY readers share one exported, read-only repeatable-read snapshot.
"""
import argparse
import gzip
import json
import re
import subprocess
import sys


def identifier(value):
    if not re.fullmatch(r"[a-z_][a-z0-9_]*", value):
        raise ValueError("Unsupported SQL identifier")
    return '"' + value + '"'


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--database", default="sub2api")
    parser.add_argument("--minutes", type=int, default=10)
    args = parser.parse_args()
    identifier(args.database)
    if not 1 <= args.minutes <= 1440:
        raise ValueError("Minutes must be between 1 and 1440")
    psql = ["sudo", "-u", "postgres", "psql", "-XqAt", "-v", "ON_ERROR_STOP=1", "-d", args.database]
    holder = subprocess.Popen(psql, stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
    try:
        holder.stdin.write("BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY; SELECT json_build_array(pg_export_snapshot(), CURRENT_TIMESTAMP::text);\n")
        holder.stdin.flush()
        snapshot, observed_at = json.loads(holder.stdout.readline())
        if not re.fullmatch(r"[A-Fa-f0-9-]+", snapshot):
            raise ValueError("Invalid exported snapshot")
        transaction = "BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY; SET TRANSACTION SNAPSHOT '" + snapshot + "'; "

        def query(sql):
            return subprocess.check_output(psql + ["-c", transaction + sql + "; COMMIT"], text=True)

        tables = json.loads(query("""SELECT COALESCE(json_agg(t),'[]') FROM (
          SELECT c.relname AS name, pg_total_relation_size(c.oid) AS bytes,
            COALESCE((SELECT json_agg(a.attname ORDER BY a.attnum) FROM pg_attribute a
              WHERE a.attrelid=c.oid AND a.attnum>0 AND NOT a.attisdropped
              AND a.attgenerated=''), '[]') AS columns
          FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
          WHERE n.nspname='public' AND c.relkind IN ('r','p') AND NOT c.relispartition
        ) t"""))
        cutoff = "TIMESTAMPTZ '" + observed_at.replace("'", "''") + "' - INTERVAL '" + str(args.minutes) + " minutes'"
        predicates = {}
        for table in tables:
            name, columns = table["name"], table["columns"]
            history = (name.endswith(("_logs", "_events", "_outbox")) or
                       name.startswith(("usage_billing_dedup", "usage_dashboard_", "ops_metrics_")) or
                       name in {"prompt_audit_jobs", "billing_usage_entries", "ops_system_metrics",
                                "ops_ingress_reject_aggregates", "usage_group_daily_rollups"})
            if not history:
                if table["bytes"] > 64 * 1024 * 1024:
                    raise ValueError("Large unclassified table requires an explicit sampling rule: " + name)
                continue
            if "bucket_date" in columns:
                predicates[name] = '"bucket_date" >= (' + cutoff + ')::date'
            elif "bucket_start" in columns:
                # Include aggregate buckets overlapping the ten-minute window.
                unit = "hour" if "hourly" in name else "minute"
                predicates[name] = '"bucket_start" >= date_trunc(\'' + unit + "', " + cutoff + ")"
            elif "created_at" in columns:
                predicates[name] = '"created_at" >= (' + cutoff + ")"
            else:
                if table["bytes"] > 64 * 1024 * 1024:
                    raise ValueError("Large historical table has no known time column: " + name)
                # Small aggregation watermarks/state are foundational data.
                continue

        # Include older referenced parents needed by recent child rows (e.g.
        # an audit event finishing a job that began before the sampling window).
        foreign_keys = json.loads(query("""SELECT COALESCE(json_agg(t),'[]') FROM (
          SELECT child.relname AS child, parent.relname AS parent,
            (SELECT json_agg(a.attname ORDER BY k.ord) FROM unnest(con.conkey) WITH ORDINALITY k(attnum,ord)
              JOIN pg_attribute a ON a.attrelid=child.oid AND a.attnum=k.attnum) AS child_columns,
            (SELECT json_agg(a.attname ORDER BY k.ord) FROM unnest(con.confkey) WITH ORDINALITY k(attnum,ord)
              JOIN pg_attribute a ON a.attrelid=parent.oid AND a.attnum=k.attnum) AS parent_columns
          FROM pg_constraint con JOIN pg_class child ON child.oid=con.conrelid
          JOIN pg_class parent ON parent.oid=con.confrelid
          JOIN pg_namespace n ON n.oid=child.relnamespace
          WHERE con.contype='f' AND n.nspname='public'
        ) t"""))

        def predicate(name, chain=()):
            if name not in predicates:
                return "TRUE"
            if name in chain:
                raise ValueError("Cyclic sampled foreign keys require an explicit rule: " + name)
            parts = [predicates[name]]
            for fk in foreign_keys:
                if fk["parent"] != name:
                    continue
                parent_columns = ",".join(map(identifier, fk["parent_columns"]))
                child_columns = ",".join(map(identifier, fk["child_columns"]))
                parts.append("(" + parent_columns + ") IN (SELECT " + child_columns +
                             " FROM public." + identifier(fk["child"]) + " WHERE " +
                             predicate(fk["child"], chain + (name,)) + ")")
            return "(" + " OR ".join(parts) + ")"

        dump = ["sudo", "-u", "postgres", "pg_dump", "-d", args.database,
                "--snapshot=" + snapshot, "--no-owner", "--no-acl"]
        print("Sample snapshot=" + snapshot + " window_minutes=" + str(args.minutes) +
              " observed_at=" + observed_at + " sampled_tables=" + str(len(predicates)), file=sys.stderr)

        with gzip.GzipFile(fileobj=sys.stdout.buffer, mode="wb", compresslevel=1, mtime=0) as output:
            def stream(command):
                process = subprocess.Popen(command, stdout=subprocess.PIPE)
                while True:
                    chunk = process.stdout.read(1024 * 1024)
                    if not chunk:
                        break
                    output.write(chunk)
                if process.wait() != 0:
                    raise RuntimeError("Database sample subprocess failed")

            stream(dump + ["--section=pre-data"])
            stream(dump + ["--data-only"] + ["--exclude-table-data-and-children=public." + name for name in predicates])
            for table in tables:
                name = table["name"]
                if name not in predicates:
                    continue
                columns = ",".join(map(identifier, table["columns"]))
                qualified = "public." + identifier(name)
                output.write(("\nCOPY " + qualified + " (" + columns + ") FROM stdin;\n").encode())
                stream(psql + ["-c", transaction + "COPY (SELECT " + columns + " FROM " + qualified +
                               " WHERE " + predicate(name) + ") TO STDOUT; COMMIT"])
                output.write(b"\\.\n")
                print("Sampled " + name, file=sys.stderr)
            stream(dump + ["--section=post-data"])
    finally:
        if holder.poll() is None:
            holder.stdin.write("ROLLBACK;\n")
            holder.stdin.close()
            holder.wait(timeout=15)


if __name__ == "__main__":
    main()
