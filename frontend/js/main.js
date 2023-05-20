import { createLoggedInHTML } from './loggedInHTML.js';
import { createMainHTML } from './mainHTML.js';
import { connectWebSocket } from './websocket.js';

window.onload = function () {
  console.log("Window loading...")
    // Check login status
    checkLoginStatus().then(loggedIn => {
      console.log("User is logged in:", loggedIn);
      // Do something based on the loggedIn status
    });
};

export function checkLoginStatus() {
  console.log("Checking login status...")
  
  return fetch('/checkLogin', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
  })
  .then(response => response.json())
  .then(data => {
    console.log("Data from checkLoginStatus:", data);
    if (data.loggedIn) {
      console.log("User is logged in.")
      console.log(data.loggedIn)
      
      createLoggedInHTML();
      connectWebSocket(data);
      return data.loggedIn;
    } else {
      console.log("User is not logged in.")
      console.log(data.loggedIn)
      
      createMainHTML();
      return data.loggedIn;
    }
  }).catch(error => {
    console.error('Error checking login status:', error);
    return false;
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