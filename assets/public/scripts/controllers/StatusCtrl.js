'use strict';

angular.module('myApp')
  .controller('StatusCtrl', function ($scope, ApiService) {
    $scope.isCollapsed = true;
    $scope.statusData = {}

    $scope.$on("fmu:update", function(ev, data) {
      $scope.statusData = data;

      $scope.link = {
        status: data.Meta.Link,
        data: parseHb(data.Hb)
      };

      $scope.flightData = {
        status: data.Meta.FlightData,
        data: {}
      }

      $scope.attCtrl = {
        status: data.Meta.AttCtrl,
        data: {}
      }

      $scope.attEst = {
        status: data.Meta.AttEst,
        data: {}
      }

      $scope.rc = {
        status: data.Meta.RC,
        data: {}
      }

      $scope.sensors = {
        status: data.Meta.Sensors,
        data: data.Imu
      }

      $scope.lpe = {
        status: data.Meta.LocalPosEst,
        data: {}
      }

      $scope.power = {
        status: data.Meta.Power,
        data: {}
      }

      $scope.globalPosEst = {
        status: data.Meta.GlobalPosEst,
        data: {}
      }

      $scope.globalPosCtrl = {
        status: data.Meta.GlobalPosCtrl,
        data: {}
      }

      // update
      $scope.$apply();
    });


    function parseHb(hb) {
      /*
      CustomMode: 65536

Type: 2

Autopilot: 12

BaseMode: 81

SystemStatus: 3

MavlinkVersion: 3
      */

      var obj = {}

      obj.Armed = (''+(!!(hb.BaseMode & 128))).toUpperCase()
      obj.ManualCtrl = (''+(!!(hb.BaseMode & 64))).toUpperCase()
      obj.SimMode = (''+(!!(hb.BaseMode & 32))).toUpperCase()
      obj.Stabilization = (''+(!!(hb.BaseMode & 16))).toUpperCase()
      obj.Guided = (''+(!!(hb.BaseMode & 8))).toUpperCase()
      obj.Auto = (''+(!!(hb.BaseMode & 4))).toUpperCase()
      obj.DevMode = (''+(!!(hb.BaseMode & 2))).toUpperCase()


      switch (hb.SystemStatus) {
        case 0: obj.status = "Unknown"; break;
        case 1: obj.status = "Booting"; break;
        case 2: obj.status = "Calibrating"; break;
        case 3: obj.status = "Standing By"; break;
        case 4: obj.status = "Active"; break;
        case 5: obj.status = "DANGER Critical Failure DANGER"; break;
        case 6: obj.status = "!! SEVERE !! EMERGENCY !! SEVERE !!"; break;
        case 7: obj.status = "Shutting Down"; break;
      }

      return obj;
    }

    // $scope.sensors = [
    //   {
    //     name: "FMU Link",
    //     status: "online"
    //   },
    //   {
    //     name: "Status",
    //     status: "online"
    //   },
    //   {
    //     name: "Sensors",
    //     status: "offline"
    //   },
    //   {
    //     name: "Power",
    //     status: "unknown"
    //   }
    // ];
  });
