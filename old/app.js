/**
 *
 * Luci Monitor
 * @author Geoff Gardner
 * @beta Early-access software.
 *
 * Copyright 2016 Dronesmith Technologies.
 */

'use strict';

var
  fs = require('fs'),
  dgram = require('dgram'),
  Emitter = require('events').EventEmitter;

var
  utils = require('./common/utils'),
  mavlinkhelper = require('./common/mavlinkhelper'),
  dronedp = require('./common/dronedp'),
  launcher = require('./common/codelauncher');

var reloader = new Emitter();
var statusMon = null;
var log = utils.log;
var settings = utils.settings;

//
// Init the app
//
reloader.on('reload', run);
reloader.emit('reload');

/**
 * Loads property data from the configuration JSON.
 *
 * @function loadProperties
 * @return {Object} loaded property data.
 * @throws {Error} an error.
 */
function loadProperties () {
  try {
    var props = fs.readFileSync(settings.PROPS_FILE);
    return JSON.parse(props);
  } catch (e) {
    throw Error('loadProps: ' + e);
  }
}

/**
 * Loads the user config file.
 *
 * @function loadConfig
 * @returns {Object} config file if successful, null otherwise.
 * @param {String} the file path to load the config file from.
 */
function loadConfig (config) {
  var stat = fs.statSync(config);

  if (stat) {
    return fs.readFileSync(config);
  } else {
    log('error', 'config not found!');
    return null;
  }
}

// (flight) Garbage collector
// TODO
// setInterval(function() {
//   var config = loadConfig();
//
//   if (config && config.tempFlightCnt) {
//     fs.stat(FORGE_SYNC, function(err, stat) {
//       if (err) {
//         console.log('[ERROR]', err.code);
//       } else {
//         console.log('[GB]', stat);
//       }
//     });
//   }
// }, 60000);

/**
 * Runs the main application. Should never exit. Gets called by the main event
 * emitter.
 *
 * @function run
 */
function run () {
  var
    client = dgram.createSocket('udp4'),
    mavConnect = dgram.createSocket('udp4');

  var
    sessionId = '',
    noSessionCnt = 0;

  var codeStatus = {
    script: null
  };

  try {
    var props = loadProperties(settings.PROPS_FILE);
  } catch (e) {
    log('error', e);
    return; // kill app
  }

  // Mavlink listener
  mavConnect.bind(props.mavlink, function () {
    mavConnect.on('message', function (msg) {
      mavlinkhelper.parseJSON(msg, function (err, result) {
        if (!err) {
          reloader.emit('system:mavlink', result);
        } else {
          log('error', err);
        }
      });
    });
  });

  // Session timer
  var sessionTimeout = setInterval(function () {
    if (noSessionCnt++ > 5) {
      sessionId = '';
      noSessionCnt = 0;
      log('warn', 'No valid reply from server.');
    }
  }, settings.SESSION_TIMER * 1000);

  // Status messages
  statusMon = setInterval(function () {
    var config = loadConfig(props.config);
    var buff;

    try {
      var cfgData = JSON.parse(config.toString());
    } catch (e) {
      log('error', e);
    }

    var sendObj = {op: 'status'};

    if (cfgData) {
      if (!cfgData.drone || sessionId === '') {
        // if no drone meta data or a session Id, send a connection request
        sendObj.email = cfgData.email;
        sendObj.password = cfgData.password;
        sendObj.serialId = cfgData.serialId;
        sendObj.op = 'connect';
        sendObj.codeStatus = codeStatus;
      }

      buff = dronedp.generateMsg(dronedp.OP_STATUS, sessionId, sendObj);
      client.send(buff, 0, buff.length, props.monitor.port, props.monitor.host);
    }
  }, settings.MONITOR_INTERVAL * 1000);

  // Echo mavlink data to host
  reloader.on('system:mavlink', function (result) {
    var buff = dronedp.generateMsg(dronedp.OP_MAVLINK_TEXT, sessionId, result);
    client.send(buff, 0, buff.length, props.monitor.port, props.monitor.host);
  });

  // Rx handling
  client.on('message', function (msg, rinfo) {
    try {
      var decoded = dronedp.parseMessage(msg);
      // Only resetting this if there was no error.
      // Might need to update this in the future, but
      // just to be safe for now.

      noSessionCnt = 0;
    } catch (e) {
      log('error', e);
    }

    // update sessionId if different.
    if (decoded.session) {
      sessionId = decoded.session;
    }

    var data = decoded.data;

    // update drone information from server.
    if (data.drone) {
      var config = loadConfig(props.config);

      try {
        var cfgData = JSON.parse(config.toString());
      } catch (e) {
        log('error', e);
      }

      cfgData.drone = data.drone;
      fs.writeFileSync(props.config, JSON.stringify(cfgData));
    }

    if (data.codeBuffer && codeStatus.script == null) {
      log('info', 'Got CODE, running job.');

      launcher.runScript(null, codeStatus, data.codeBuffer, settings.CODE_EXEC);
    }
  });

  launcher.getEmitter().on('code:update', function (msg) {
    var buff = dronedp.generateMsg(dronedp.OP_STATUS, sessionId,
      {op: 'code', msg: msg, status: codeStatus});
    client.send(buff, 0, buff.length, props.monitor.port, props.monitor.host);
  });

  // Reinitialize app on error
  client.on('error', function (err) {
    client.close();

    log('error', err);

    setTimeout(function () {
      reloader.emit('reload');
    }, settings.RELOAD_TIME * 1000);
  });
}
