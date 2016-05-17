'use strict';

angular.module('myApp')
  .controller('StatusCtrl', function ($scope, $interval, $window, ApiService) {
    $scope.isCollapsed = true;
    $scope.statusData = {}
    $scope.noConnect = false;
    $scope.outputs = [];

    var timeOutCnt = 0;

    ApiService.initSocket();

    $scope.API = ApiService;

    $scope.logout = function() {

      $scope.API.logout(function(res) {
        $window.location.replace("/");
      })
    };

    $scope.getOutputs = function() {
      $scope.API.getOutputs(function(res) {
        $scope.outputs = res.data.Outputs;
      });
    }

    $scope.addOutput = function(out) {
      $scope.API.addOutput(out, function(data) {
        $scope.getOutputs();
      });
    }

    $scope.delOutput = function(out) {
      $scope.API.removeOutput(out, function(data) {
        $scope.getOutputs();
      });
    }

    $scope.getOutputs();

    $scope.$on("fmu:update", function(ev, data) {
      $scope.statusData = data;
      $scope.noConnect = false;
      timeOutCnt = 0;

      $scope.cloud = {
        status: data.CloudOnline || "offline"
      }

      $scope.link = {
        status: data.Meta.Link || "offline",
        data: parseHb(data.Hb)
      };

      $scope.flightData = {
        status: data.Meta.FlightData || "offline",
        data: data.Vfr
      }

      $scope.attCtrl = {
        status: data.Meta.AttCtrl || "offline",
        data: data.AttCtrl
      }

      $scope.attEst = {
        status: data.Meta.AttEst || "offline",
        data: data.AttEst
      }

      $scope.rc = {
        status: data.Meta.RC || "offline",
        data: parseRc(data.RcValues, data.RcStatus)
      }

      $scope.sensors = {
        status: data.Meta.Sensors || "offline",
        data: data.Imu
      }

      $scope.lpe = {
        status: data.Meta.LocalPosEst || "offline",
        data: data.LocalPos
      }

      $scope.power = {
        status: data.Meta.Power || "offline",
        data: parseBattery(data.Battery)
      }

      // determined by number of satellites visible
      $scope.globalPosEst = {
        status: data.Gps.SatellitesVisible ? data.Meta.GlobalPosEst : 'offline',
        data: parseGps(data.GlobalPos, data.Gps)
      }

      $scope.globalPosCtrl = {
        status: data.Meta.GlobalPosCtrl || "offline",
        data: data.GlobalPosTarget
      }

      $scope.gps = {
        status: data.Meta.Gps || "offline",
        data: data.Gps
      }

      $scope.servos = {
        status: data.Meta.Servos || "offline",
        data: parseServos(data.Servos, data.Actuators)
      }

      $scope.alts = {
        status: data.Meta.Altitude || "offline",
        data: data.Altitude
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
        $scope.alts.status = "offline";
        $scope.servos.status = "offline";
        $scope.statusData = {};
      }
    }, 1000);

    function parseServos(motors, target) {
      var obj = {}

      obj.servos = [];
      obj.targets = [];

      for (var i = 0; i < target.Controls.length; ++i) {
        var val = target.Controls[i];
        if (val != 0) {
          obj.targets.push(val);
          obj.servos.push(motors['Servo' + (i+1) + 'Raw']);
        }
      }

      return obj;
    }


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
