var
  fs = require('fs'),
  path = require('path'),
  console = require('console');

exports.settings = {
  PROPS_FILE : 'properties.json',
  CODE_EXEC : 'exec.py',
  CODE_LAUNCHER : 'python',
  RELOAD_TIME : 10,
  MONITOR_INTERVAL : 1,
  SESSION_TIMER : 5,
  MAV_SYSTEM: 1,
  MAV_COMPONENT: 1
};

// Init the logger
var output = fs.createWriteStream(path.resolve('./lucimon.log'));
var logger = new console.Console(output, output);


/**
 * Simple function for logging data.
 *
 * @function log
 */
function log(level, str) {
  console[level]('[MON]', str);
  logger[level](new Date() + ': ' + str);
}

exports.log = log;
