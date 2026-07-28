import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import batchImage from './batchImage'
import leaderboard from './leaderboard'
import community from './community'
import admin from './admin'
import misc from './misc'

export default {
  ...landing,
  ...common,
  ...dashboard,
  ...batchImage,
  ...leaderboard,
  ...community,
  admin,
  ...misc,
}
