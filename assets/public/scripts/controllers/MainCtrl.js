'use strict';

angular.module('myApp')
  .controller('MainCtrl', function ($scope, $window, ApiService) {
    $scope.toggle = false;
    $scope.aps = ApiService.aps;
    $scope.email = "";
    $scope.password = "";

    $scope.submitted = false;
    $scope.responseError = "";

    $scope.submit = function(){
      $scope.responseError = "";
      if ($scope.email != "" && $scope.password != "") {
        $scope.submitted = true;
        ApiService.setUp($scope.email, $scope.password, function(data) {
          $scope.submitted = false;
          if (data.error) {
            $scope.responseError = data.error;
          } else {
            $scope.responseError = "Success! Please refresh this page.";
            $window.location.reload();
          }
        });
      }
    };
  });
