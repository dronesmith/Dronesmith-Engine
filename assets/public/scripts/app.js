'use strict';

angular.module('myApp', ['ngRoute', 'btford.socket-io', 'ui.bootstrap'])
  .config(function ($routeProvider, $locationProvider) {
    $routeProvider
      .when('/', {
        templateUrl: 'views/' + DEFAULT_ROUTE,
        controller: DEFAULT_CONTROL
      })
      .when('/status',{
        templateUrl: 'views/status.html',
        controller: 'StatusCtrl'
      })
      .when('/wifi',{
        templateUrl: 'views/wifi.html',
        controller: 'WifiCtrl'
      })
      .otherwise({
        templateUrl: '404.html'
      });

    $locationProvider.html5Mode(true);
  });
