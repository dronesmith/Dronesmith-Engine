'use strict';

angular.module('myApp')
  .controller('StatusCtrl', function ($scope, $interval, ApiService) {
    $scope.isCollapsed = true;
    $scope.statusData = {}
    $scope.noConnect = false;

    var timeOutCnt = 0;

    ApiService.initSocket();

    $scope.$on("fmu:update", function(ev, data) {
      $scope.statusData = data;
      $scope.noConnect = false;
      timeOutCnt = 0;

      $scope.link = {
        status: data.Meta.Link,
        data: parseHb(data.Hb)
      };

      $scope.flightData = {
        status: data.Meta.FlightData,
        data: data.Vfr
      }

      $scope.attCtrl = {
        status: data.Meta.AttCtrl,
        data: data.AttCtrl
      }

      $scope.attEst = {
        status: data.Meta.AttEst,
        data: data.AttEst
      }

      $scope.rc = {
        status: data.Meta.RC,
        data: parseRc(data.RcValues, data.RcStatus)
      }

      $scope.sensors = {
        status: data.Meta.Sensors,
        data: data.Imu
      }

      $scope.lpe = {
        status: data.Meta.LocalPosEst,
        data: data.LocalPos
      }

      $scope.power = {
        status: data.Meta.Power,
        data: parseBattery(data.Battery)
      }

      // determined by number of satellites visible
      $scope.globalPosEst = {
        status: data.Gps.SatellitesVisible ? data.Meta.GlobalPosEst : 'offline',
        data: parseGps(data.GlobalPos, data.Gps)
      }

      $scope.globalPosCtrl = {
        status: data.Meta.GlobalPosCtrl,
        data: data.GlobalPosTarget
      }

      $scope.gps = {
        status: data.Meta.Gps,
        data: data.Gps
      }

      // update
      $scope.$apply();
    });

    $interval(function() {
      timeOutCnt++

      if (timeOutCnt > 5) {
        $scope.noConnect = true;

        $scope.link.status = "offline";
        $scope.flightData.status = "offline";
        $scope.attCtrl.status = "offline";
        $scope.attEst.status = "offline";
        $scope.rc.status = "offline";
        $scope.sensors.status = "offline";
        $scope.lpe.status = "offline";
        $scope.power.status = "offline";
        $scope.globalPosEst.status = "offline";
        $scope.globalPosCtrl.status = "offline";
        $scope.gps.status = "offline";
      }
    }, 1000);


    function parseHb(hb) {

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

    function parseRc(rc, rs) {
      var obj = {};

      obj['Signal Strength (RSSI)'] = rs.Rssi;

      for (var i = 1; i <= rc.Chancount; ++i) {
        var key = 'Chan'+i+'Raw';
        obj['Channel ' + i + ' PWM'] = rc[key];
      }

      return obj
    }

    function parseBattery(bat) {
      var obj = {};

      switch (bat.BatteryFunction) {
        case 0: obj.purpose = "Unknown"; break;
        case 1: obj.purpose = "All Flight Systems"; break;
        case 2: obj.purpose = "Propulsion Systems"; break;
        case 3: obj.purpose = "Avionics"; break;
        case 4: obj.purpose = "Payloads"; break;
      }

      switch (bat.Type) {
        case 0: obj.type = "Unknown"; break;
        case 1: obj.type = "Lithium Polymer (LIPO)"; break;
        case 2: obj.type = "Lithium Iron Phosphate (LIFE)"; break;
        case 3: obj.type = "Lithium Ion (LION)"; break;
        case 4: obj.type = "Nickel Metal Hydride (NIMH)"; break;
      }

      obj.remaining = bat.BatteryRemaining;

      return obj;
    }

    function parseGps(glb, gps) {
      return {
        lat: glb.Lat,
        lon: glb.Lon,
        relativeAlt: glb.RelativeAlt,
        velx: glb.Vx,
        vely: glb.Vy,
        velz: glb.Vz,
        sats: gps.SatellitesVisible
      };
    }
  });
