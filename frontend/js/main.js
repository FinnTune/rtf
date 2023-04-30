import { login } from './login.js';
import { register } from './register.js';
import { createLoggedInHTML } from './loggedInHTML.js';
import { createMainHTML } from './mainHTML.js';

window.onload = function () {
    // Check login status
    // checkLoginStatus();

    document.getElementById('login-form').addEventListener('submit', function(event) {
        event.preventDefault();
        login();
    });
    document.getElementById('registration-form').addEventListener('submit', function(event) {
        event.preventDefault();
        register();
    });
};


function checkLoginStatus() {
    // Commented is unnecessary code as cookie gets sent automatically with request
    // const sessionID = getCookie('sessionID'); // Replace 'sessionID' with the name of your session cookie
  
    // Make an AJAX request to the backend to check if the session ID is valid
    fetch('/checkLogin', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
    //   body: JSON.stringify({ sessionID })
    })
    .then(response => response.json())
    .then(data => {
      if (data.loggedIn) {
        // User is logged in
        createLoggedInHTML();
      } else {
        // User is not logged in
        createMainHTML();
      }
    })
    .catch(error => {
      console.error('Error checking login status:', error);
    });
  }
  
  // Helper function to get the value of a cookie
//   function getCookie(name) {
//     const value = `; ${document.cookie}`;
//     const parts = value.split(`; ${name}=`);
//     if (parts.length === 2) {
//       return parts.pop().split(';').shift();
//     }
//   }