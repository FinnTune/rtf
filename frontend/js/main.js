import {login} from './login.js';

window.onload = function () {
    document.getElementById('login-form').addEventListener('submit', function(event) {
        event.preventDefault();
        login();
        document.getElementById('login-form').style.display = 'none';
        document.getElementById('intro').innerHTML = 'You are now logged in.';
        document.getElementById('intro').style.display = 'block';
        document.getElementById('main-content').style.display = 'block';
    });
};
