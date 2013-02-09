function StaticCtrl($scope) {

}

function PlayCtrl($scope, $http) {
  $http.get('/data/games.json').success(function(data) {
      $scope.games = data;
  });

}

function UserCtrl($scope) {
  $scope.isLoggedIn = false;
}