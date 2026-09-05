import overview from './overview'
import balances from './balances'
import channels from './channels'
import accounts from './accounts'
import resources from './resources'
import ops from './ops'
import settings from './settings'
import audit from './audit'
import promptAudit from './promptAudit'
import plugins from './plugins'

export default {
  ...overview,
  ...balances,
  ...channels,
  ...accounts,
  ...resources,
  ...ops,
  ...settings,
  ...audit,
  ...promptAudit,
  ...plugins,
}
