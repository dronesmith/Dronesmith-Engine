'use strict';

var
  cp = require('child_process'),
  path = require('path'),
  events = require('events');

  var
    utils = require('./utils'),
    log = utils.log,
    settings = utils.settings,
    emitter = new events.EventEmitter();

exports.runScript = function (id, session, snippet, customPath) {
  var script;
  var codePath;

  if (customPath) {
    codePath = customPath;
  } else {
    codePath = path.resolve(settings.CODE_EXEC);
  }

  if (snippet) {
    script = cp.spawn(settings.CODE_LAUNCHER,
      [codePath, '--code', snippet]);
  } else {
    script = cp.spawn(settings.CODE_LAUNCHER,
      [codePath, '--id', id]);
  }

  session.script = script.pid;

  log('info', 'running from ' + session.script);
  emitter.emit('code:update', 'Running job ' + session.script + '...');
  // script.disconnect();

  script.on('close', function (code) {
    if (!isNaN(code)) {
      log('info', 'Process ended ' + code);
      emitter.emit('code:update', 'App ended with exit code ' + code + '.');
    } else {
      log('warn', 'Process ended abnormally from SIG ' + code);
    }
    session.script = null;
  });

  script.on('error', function (err) {
    log('error', err);
    // script.disconnect();
    session.script = null;
  });

  script.stdout.on('data', function (data) {
    log('trace', 'STDOUT ' + data.toString());
    emitter.emit('code:update', data.toString());
  });

  script.stderr.on('data', function (data) {
    log('trace', 'STDERR ' + data.toString());
    emitter.emit('code:update', data.toString());
    session.script = 'error';
  });
};

exports.getEmitter = function () {
  return emitter;
};
