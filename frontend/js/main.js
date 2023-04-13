import {login} from './login.js';

window.onload = function () {
    document.getElementById('login-form').addEventListener('submit', function(event) {
        event.preventDefault();
        login();
    });
};
