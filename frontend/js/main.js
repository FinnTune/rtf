import {login} from './login.js';
import {register} from './register.js';

window.onload = function () {
    document.getElementById('login-form').addEventListener('submit', function(event) {
        event.preventDefault();
        login();
        document.getElementById('login-form').style.display = 'none';
        document.getElementById('main-content').style.display = 'block';
    });
    document.getElementById('registration-form').addEventListener('submit', function(event) {
        event.preventDefault();
        register();
        document.getElementById('login-form').style.display = 'none';
        document.getElementById('main-content').style.display = 'block';
    });
};
