import { createMainHTML } from "./mainHTML.js";

export function logout () {
    fetch('/logout', {
        method: 'POST',
        mode: 'cors',
        headers: {
          'Content-Type': 'application/json'
        },
      //   body: JSON.stringify({ sessionID })
      })
      .then(response => response.json())
      .then(data => {
        if (data.loggedIn == false) {
          console.log("User is logged out.")
          console.log(data.loggedIn)
          
          // User is logged in
          createMainHTML();
        } else {
          console.log("User logout failed.")
          console.log(data.loggedIn)
          // User is not logged in
          createMainHTML();
        }
      })
      .catch(error => {
        console.error('Error checking login status:', error);
      });
}