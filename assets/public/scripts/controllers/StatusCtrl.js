'use strict';

angular.module('myApp')
  .controller('StatusCtrl', function ($scope, ApiService) {
    $scope.isCollapsed = true;
    $scope.statusData = {}

    $scope.$on("fmu:update", function(ev, data) {
      $scope.statusData = data;
      $scope.$apply();
    });

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
