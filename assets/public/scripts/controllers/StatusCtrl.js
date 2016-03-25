'use strict';

angular.module('myApp')
  .controller('StatusCtrl', function ($scope,ApiService) {
    $scope.isCollapsed = true;

    ApiService.onStatusUpdate = function(event) {
      $scope.statusData = event.data
    };

    $scope.sensors = [
      {
        name: "FMU Link",
        status: "online"
      },
      {
        name: "Status",
        status: "online"
      },
      {
        name: "Sensors",
        status: "offline"
      },
      {
        name: "Power",
        status: "unknown"
      }
    ];
  });
