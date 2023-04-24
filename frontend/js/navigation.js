//Event listeners for navigation and home buttons
document.getElementById('title').addEventListener('click', function() {
    document.getElementById('login-form').style.display = 'none';
    document.getElementById('intro').style.display = '';
    document.getElementById('main-content').style.display = 'block';
    document.getElementById('registration-form').style.display = 'none';
});

document.getElementById('register-button').addEventListener('click', function() {
    document.getElementById('login-form').style.display = 'none';
    document.getElementById('intro').style.display = 'none';
    document.getElementById('main-content').style.display = 'none';
    document.getElementById('registration-form').style.display = 'block';
});

document.getElementById('login-button').addEventListener('click', function() {
    document.getElementById('login-form').style.display = 'block';
    document.getElementById('intro').style.display = 'none';
    document.getElementById('main-content').style.display = 'none';
    document.getElementById('registration-form').style.display = 'none';
});

document.getElementById('login-switch-button').addEventListener('click', function() {
    document.getElementById('login-form').style.display = 'block';
    document.querySelector('.register-form').style.display = 'none';
});

document.getElementById('register-switch-button').addEventListener('click', function() {
    document.getElementById('login-form').style.display = 'none';
    document.querySelector('.register-form').style.display = 'block';
});