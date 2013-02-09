function PlayCtrl($scope, $location, $http) {
	$scope.loggedIn = false;

	$scope.refreshRooms = function() {   
		$http.get("/api/rooms").
			success(function(data, status, headers, config) {
				console.log("s",data,status,headers,config);
				$scope.loggedIn = true;
				$scope.rooms = data;
				console.log($scope.rooms);
			}).
			error(function(data, status, headers, config) {
				if (status == 403) {
					$scope.loggedIn = false;
				}
				console.log("e",data,status,headers,config);
				// pause refreshing
				$scope.revealLogin();
			});
	};

	$scope.makeRoom = function() {
		alert($scope.makeroomform);
		$http.post("/api/rooms", $scope.makeroomform).
			success(function(data, status, headers, config) {
				console.log("s",data,status,headers,config);
			}).
			error(function(data, status, headers, config) {
				console.log("e",data,status,headers,config);
			});
	};

	$scope.revealMakeRoom = function() { $("#makeroom").reveal(); }
	$scope.revealLogin = function() { $("#login").reveal(); };

	$scope.login = function() {
		$http.post("/api/login", $scope.loginform).
			success(function(data, status, headers, config) {
				console.log("s",data,status,headers,config);
			}).
			error(function(data, status, headers, config) {
				console.log("e",data,status,headers,config);
			});
	};

	$scope.register = function() {
		$http.post("/api/register", $scope.registerform).
			success(function(data, status, headers, config) {
				console.log("s",data,status,headers,config);
			}).
			error(function(data, status, headers, config) {
				console.log("e",data,status,headers,config);
			});
	};

	$scope.changeRoom = function(roomId) {
		$http.get("/api/cmd/changeroom?roomId="+roomId).
			success(function(data, status, headers, config) {
				console.log("s",data,status,headers,config);
			}).
			error(function(data, status, headers, config) {
				console.log("e",data,status,headers,config);
			});
			// make them wait a few seconds before giving up or truly
			// switching
	};

	setTimeout($scope.refreshRooms, 1000);
	//$scope.refreshRooms();
}