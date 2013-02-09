var xbtunnel = angular.module('xbtunnel', []);

xbtunnel.config(function($routeProvider, $locationProvider) {
    $locationProvider.html5Mode(true);

    $routeProvider.
      when('/home',                 {templateUrl: 'partials/home.html',    controller: StaticCtrl}).
      when('/faq',                  {templateUrl: 'partials/faq.html',     controller: StaticCtrl}).
      when('/help',                 {templateUrl: 'partials/help.html',    controller: StaticCtrl}).

      when('/play',                 {templateUrl: 'partials/play.html',    controller: PlayCtrl}).

      when('/login',                {templateUrl: 'partials/login.html',   controller: UserCtrl}).
      when('/profile',              {templateUrl: 'partials/profile.html', controller: StaticCtrl}).

      otherwise({redirectTo: '/home'});
});