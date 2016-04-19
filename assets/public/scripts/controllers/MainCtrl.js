'use strict';

angular.module('myApp')
  .controller('MainCtrl', function ($scope, $window, ApiService) {
    $scope.toggle = false;
    $scope.email = "";
    $scope.password = "";
    $scope.name = "";
    $scope.username = "";

    $scope.submitted = false;
    $scope.responseError = "";

    ApiService.getSetupStage(function(data) {
      $scope.SetupStep = data.step;

      if ($scope.SetupStep == 1) {
        ApiService.getNetworks(function(data) {
          $scope.aps = data.aps;
          $scope.netdata = {};
          $scope.netdata.ap = Object.keys($scope.aps)[0];
        });
      }
    });

    $scope.submitStep1 = function(netdata) {
      $scope.responseError = "";
      $scope.submitted = true;

      ApiService.activateNetwork({
        name: netdata.name,
        ssid: netdata.ap,
        protocol: $scope.aps[netdata.ap],
        password: netdata.password,
        username: netdata.username
      }, function(data) {
        $scope.submitted = false;
        if (data.error) {
          $scope.responseError = data.error;
        } else {
          $scope.responseError = "Success! Luci is now connected to your local area network. Please connect to `"
            + data.ssid + "` and go to `http://" + data.name + ".local` (" + data.ip + ") to continue the setup process.";
          $scope.HideConnect = true;
        }
      });
    };

    $scope.submit = function(){
      $scope.responseError = "";
      if ($scope.email != "" && $scope.password != "") {
        $scope.submitted = true;
        ApiService.setUp($scope.email, $scope.password, function(data) {
          $scope.submitted = false;
          if (data.error) {
            $scope.responseError = data.error;
          } else {
            // $scope.responseError = "Success! Please refresh this page.";
            // $window.location.reload();

            $scope.SetupStep++;
          }
        });
      }
    };
  });
