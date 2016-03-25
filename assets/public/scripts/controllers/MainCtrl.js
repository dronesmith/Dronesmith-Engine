'use strict';

angular.module('myApp')
  .controller('MainCtrl', function ($scope,ApiService) {
    $scope.toggle = false;
    $scope.aps = ApiService.aps;
    $scope.email = "";
    $scope.password = "";
    $scope.submit = function(){
      if ($scope.email!=""&&$scope.password!="") {
            ApiService.setUp(btoa($scope.email),btoa($scope.password));
            $scope.toggle = true;
      }
    };
  });
