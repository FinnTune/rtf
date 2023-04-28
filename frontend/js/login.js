import { connectWebSocket } from './websocket.js';

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

    // console.log(loginFormData);
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
            document.getElementById('msg').innerHTML = 'You are now logged in.';
            document.getElementById('msg').style.display = "block"
            document.getElementById('all-posts-button').style.display = "block"
            document.getElementById('create-post-button').style.display = "block"
            document.getElementById('login-button').style.display = "none"
            document.getElementById('logout-button').style.display = "block"
            document.getElementById('register-button').style.display = "none"
            document.getElementById('intro').style.display = 'flex';
            document.getElementById('login-form').style.display = 'none';
            document.getElementById('main-content').style.display = 'block'; 
            document.getElementById('chat').style.display = "block";
            return response.json();
        } else {
            // throw new Error('Unauthorized');
            throw 'Unauthorized';
        }
    }).then((data) => {
        //At this point user is authenticated
        connectWebSocket(data.otp);
    }).catch((error) => {
        alert("Err: " + error);
        console.log("Err: ", error);
    });
 
    return false;
}