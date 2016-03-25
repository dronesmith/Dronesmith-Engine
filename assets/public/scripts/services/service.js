'use strict';
angular.module('myApp')
  .service('ApiService', function ($http, $rootScope, $log) {
  this.uid = null;
  this.socket = new WebSocket(SOCKET_ADDRESS);

  this.aps = [
    {
      ssid: "test1",
      kind: "Wi-fi"
    },
    {
      ssid: "test2",
      kind: "LTE"
    }
  ];

  this.setUp = function (user,pwd) {
    var data = JSON.stringify({
              email: user,
              password: pwd
          });

    $http.post("/api/setup", data).success(function(data, status) {
            this.uid = data.id;
            $http.get("/api/aps").success(function (data, status) {
                this.aps = data.aps;
            });
       });
  };

    this.setAsp = function (access) {
    };

    // Send status update to all scopes
    this.socket.onmessage = function(event) {
      try {
        var json = JSON.parse(event.data)
      } catch (e) {
        $log.error(e)
      }
      $rootScope.$broadcast("fmu:update", json);
    };

  });
