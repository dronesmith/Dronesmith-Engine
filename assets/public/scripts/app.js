'use strict';

angular.module('myApp', ['ngRoute','ui.bootstrap'])
  .config(function ($routeProvider) {
    $routeProvider
      .when('/', {
        templateUrl: 'views/main.html',
        controller: 'MainCtrl'
      })
      .when('/status',{
        templateUrl: 'views/status.html',
        controller: 'StatusCtrl'
      })
      .otherwise({
        redirectTo: '/'
      });
  });
