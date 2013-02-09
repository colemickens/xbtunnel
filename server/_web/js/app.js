var xbtunnel = angular.module('xbtunnel', []);

xbtunnel.config(function($routeProvider, $locationProvider) {
	//$locationProvider.html5Mode(true);

	$routeProvider.
		when('/',      { templateUrl: '/partials/home.html',  controller: HomeCtrl   }).
		when('/play',  { templateUrl: '/partials/play.html',  controller: PlayCtrl   }).

		otherwise({redirectTo: '/'});
});

angular.element(document).ready(function() { 
	$(document).foundationNavigation();
	$(document).foundationTopBar();
	$(document).foundationCustomForms();
});