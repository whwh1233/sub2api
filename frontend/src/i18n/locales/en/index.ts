import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import community from './community'
import leaderboard from './leaderboard'
import admin from './admin'
import misc from './misc'

export default {
  ...landing,
  ...common,
  ...dashboard,
  ...community,
  ...leaderboard,
  admin,
  ...misc,
}
