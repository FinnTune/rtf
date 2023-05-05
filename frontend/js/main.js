import { createLoggedInHTML } from './loggedInHTML.js';
import { createMainHTML } from './mainHTML.js';
import { connectWebSocket } from './websocket.js';

window.onload = function () {
  console.log("Window loaded.")
    // Check login status
    checkLoginStatus();
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
        console.log(data.loggedIn)
        
        // User is logged in
        createLoggedInHTML();
        connectWebSocket(data.otp);
      } else {
        console.log("User is not logged in.")
        console.log(data.loggedIn)
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