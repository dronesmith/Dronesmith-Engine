'use strict';

var
  utils = require('./utils'),
  settings = utils.settings,
  events = require('events'),
  Mavlink = require('mavlink');

// Init
var mav = new Mavlink(settings.MAV_SYSTEM, settings.MAV_COMPONENT);
var emitter = new events.EventEmitter();

// Set up listeners
mav.on('ready', function () {
  mav.on('messasge', function (msg) {
    emitter.emit('message', msg);
  });

  // All json message events reflected. Ensures we get all messages coming.
  emitter.setMaxListeners(Object.keys(mav.messagesByName).length);
  for (var k in mav.messagesByName) {
    mav.on(k, function (msg, data) {
      emitter.emit('jsonmessage', {header: mav.getMessageName(msg.id), data: data});
    });
  }

  mav.on('error', function (err) {
    emitter.emit('error', err);
  });
});

exports.parseBinary = function (buffer, cb) {
  mav.parse(buffer);

  var handleBin = function (msg) {
    emitter.removeListener('message', handleBin);
    return cb(null, msg);
  };

  var handleError = function (err) {
    emitter.removeListener('error', handleError);
    return cb(err);
  };

  emitter.on('message', handleBin);
  emitter.on('error', handleError);
};

exports.parseJSON = function (buffer, cb) {
  mav.parse(buffer);

  var handleJSON = function (msg) {
    emitter.removeListener('jsonmessage', handleJSON);
    return cb(null, msg);
  };

  var handleError = function (err) {
    emitter.removeListener('error', handleError);
    return cb(err);
  };

  emitter.on('jsonmessage', handleJSON);
  emitter.on('error', handleError);
};
