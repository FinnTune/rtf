import { createLoggedInHTML } from './loggedInHTML.js';
import { createMainHTML } from './mainHTML.js';

window.onload = function () {
  console.log("Window loaded.")
    // Check login status
    checkLoginStatus();

    // Add event listeners to login and registration forms
    // if (document.getElementById('login-form') != null) {
    // document.getElementById('login-form').addEventListener('submit', function(event) {
    //     event.preventDefault();
    //     login();
    // });
    // }
    // if (document.getElementById('registration-form') != null) {
    // document.getElementById('registration-form').addEventListener('submit', function(event) {
    //     event.preventDefault();
    //     register();
    // });
    // }
    // This onsubmit uses the form to reload the page and the websocket gets reloaded which is why you cant see the message
    // document.getElementById('new-message').onsubmit = sendMessage;
    // if (document.getElementById('chat') != null) {
    //   document.getElementById('chat').addEventListener('submit', function(event) {
    //     event.preventDefault();
    //     sendMessage();
    //   });
    // }
    // if (document.getElementById('logout') != null) {
    //   document.getElementById('logout').addEventListener('click', function(event) {
    //     event.preventDefault();
    //     logout();
    //   });
    // }
};


function checkLoginStatus() {
  console.log("Checking login status...")
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
        console.log("User is logged in.")
        // User is logged in
        createLoggedInHTML();
      } else {
        console.log("User is not logged in.")
        // User is not logged in
        createMainHTML();
      }
    })
    .catch(error => {
      console.error('Error checking login status:', error);
    });
  }
  
  //ChatGPT suggestion for grabbing cookie with JS!!!
  // Helper function to get the value of a cookie
//   function getCookie(name) {
//     const value = `; ${document.cookie}`;
//     const parts = value.split(`; ${name}=`);
//     if (parts.length === 2) {
//       return parts.pop().split(';').shift();
//     }
//   }