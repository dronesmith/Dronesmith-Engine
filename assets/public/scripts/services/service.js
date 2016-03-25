'use strict';
angular.module('myApp')
  .service('ApiService', function ($http) {
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

    this.onStatusUpdate = this.socket.onmessage;

  });
