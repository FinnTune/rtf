import { connectWebSocket } from './websocket.js';

export function register() {
     //Check if password and confirm password are the same
     let password = document.getElementById('regpassword').value;
     console.log(password);
     let confirmPassword = document.getElementById('regconfpassword').value;
     console.log(confirmPassword);
     if(password != confirmPassword){
         alert("Passwords do not match");
         return false;
     }
    //Using document.getElementById for specific fields
    let formData = {
        fname: document.getElementById('regfname').value,
        lname: document.getElementById('reglname').value,
        uname: document.getElementById('reguname').value,
        email: document.getElementById('regemail').value,
        age: document.getElementById('regage').value,
        gender: document.getElementById('reggender').value,
        password: document.getElementById('regpassword').value,
    };

    // console.log(loginFormData);
    console.log(formData);

    fetch('register', {
        method: 'POST',
        body: JSON.stringify(formData),
        mode: 'cors',
        headers: {
            'Content-Type': 'application/json'
        }}
    ).then((response) => {
        if(response.ok){
            document.getElementById('intro').innerHTML = 'You are now registered. Please login.';
            document.getElementById('intro').style.display = 'block';
            return;
        } else {
            // throw new Error('Unauthorized');
            throw 'Unauthorized';
        }
    }).catch((error) => {
        alert("Err: " + error);
        console.log("Err: ", error);
    });
 
    return false;
}