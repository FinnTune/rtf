import { connectWebSocket } from './websocket.js';
import { createLoggedInHTML } from './loggedInHTML.js';
import { getAllPosts } from './getAllPosts.js';
import { checkLoginStatus } from './main.js';

export function login() {
    //Below are two different was to get the form data

    //Using new FormData dircetly into JSON.stringify does not work without the following two lines.
    // let loginFormData = new FormData(document.getElementById('login-form'));
    // let formDataArray = Array.from(loginFormData.entries());
    // let formDataJson = Object.fromEntries(formDataArray);

    //Using document.getElementById for specific fields
    let formData = {
        username: document.getElementById('username').value,
        password: document.getElementById('password').value,
    };

    checkLoginStatus().then(loggedIn => {
        if (loggedIn) {
            console.log("User is already logged in.")
            return;
        } else {
            console.log(formData);

            fetch('login', {
                method: 'POST',
                body: JSON.stringify(formData),
                mode: 'cors',
                headers: {
                    'Content-Type': 'application/json'
                }}
            ).then((response) => {
                if(response.ok){
                console.log("User logged in.")
                // User is logged in
                createLoggedInHTML();
                    return response.json();
                } else {
                    // throw new Error('Unauthorized');
                    throw 'Unauthorized';
                }
            }).then((data) => {
                console.log("Data from login:", data)
                //Save data in local storage
                localStorage.setItem('id', data.id);
                localStorage.setItem('username', data.username);
                localStorage.setItem('email', data.email);
                localStorage.setItem('joined', data.joined);
                //At this point user is authenticated
                connectWebSocket(data);
                document.getElementById('msg').innerHTML = data.username + ', you are now logged in.';
                getAllPosts();
            }).catch((error) => {
                alert("Err: " + error);
                console.log("Err: ", error);
            });
        
            return false;
        }
    });
};
