'use strict';
angular.module('myApp')
  .service('ApiService', function ($http, $rootScope, $log) {
  this.uid = null;

  this.apiError = null;

  this.setUp = function (user,pwd, cb) {
    var data = JSON.stringify({ email: user, password: pwd});

    $http.post("/index/setup", data).success(function(data, status) {
      return cb(data);
       });
  };

  this.getSetupStage = function(cb) {
    $http.get("/index/setup").success(function(data, status) {
      return cb(data);
    });
  }

  this.getNetworks = function(cb) {
    $http.get("/index/aps")
      .success(function(data, status) {
      return cb(data);
    })
    ;
  }

  this.activateNetwork = function(data, cb) {
    $http.post("/index/aps", data)
      .success(function(data, status) {
        return cb(data);
      })
  }

    this.setAsp = function (access) {
    };

    this.initSocket = function() {
      if (this.socket) {
        this.socket.close();
      }

      this.socket = _initSock();
    };

    this.addOutput = function(str, cb) {
      $http.post('/index/output', {"Address": str}).then(cb);
    }

    this.removeOutput = function(str, cb) {
      $http.post('/index/output', {Method: "delete", "Address": str}).then(cb);
    }

    this.getOutputs = function(cb) {
      $http.get('/index/output').then(function(response) {
        return cb(response);
      })
    }

    this.sendBind = function(b, cb) {
      $http.post('/index/bind', {"Proto": +b}).then(cb);
    }

    this.logout = function(cb) {
      $http
        .post('/index/logout', {})
        .then(function(response) {
          return cb(response.data);
        })
      ;
    }

    function _initSock() {
      try {
        var socket = new WebSocket(SOCKET_ADDRESS);
      } catch(e) {
        console.log(e);
        return;
      }

      socket.onclose = function(event) {
        socket = null;
        _initSock();
      };

      socket.onerror = function(event) {
        console.log("error");
        // console.log(event);
      };

      // Send status update to all scopes
      socket.onmessage = function(event) {
        try {
          var json = JSON.parse(event.data)
        } catch (e) {
          $log.error(e)
        }
        $rootScope.$broadcast("fmu:update", json);
      };

      return socket;

    }

  })
  .factory('apiSocket', function (socketFactory) {
    return socketFactory();
  })
  ;
