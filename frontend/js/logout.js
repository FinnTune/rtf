import { createMainHTML } from "./mainHTML.js";

export async function logout () {
    console.log("Logging out...")
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
          document.getElementById('msg').innerHTML = "You've been logged out."
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