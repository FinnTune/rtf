// Wait for the document to be fully loaded before executing the code
document.addEventListener('DOMContentLoaded', function() {

    // Check if user is logged in on page load
    checkLoggedIn();
  
    // Add event listener to the login form
    document.getElementById('login-form').addEventListener('submit', function(e) {
      e.preventDefault(); // prevent the default form submission
  
      // Send the form data to the server using AJAX
      var xhr = new XMLHttpRequest();
      xhr.open('POST', '/login');
      xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
      xhr.onload = function() {
        if (xhr.status === 200) {
          // Handle the server response here
          checkLoggedIn(); // check if user is logged in after login attempt
        } else {
          // Handle any errors here
        }
      };
      xhr.send(new FormData(this));
    });
  
    // Add event listener to the register button
    document.getElementById('register-button').addEventListener('click', function() {
      document.getElementById('login-form').style.display = 'none';
      document.querySelector('.register-form').style.display = 'block';
    });
  
    // Add event listener to the login switch button
    document.getElementById('login-switch-button').addEventListener('click', function() {
      document.getElementById('login-form').style.display = 'block';
      document.querySelector('.register-form').style.display = 'none';
    });
  
    // Add event listener to the register form
    document.getElementById('register-form').addEventListener('submit', function(e) {
      e.preventDefault(); // prevent the default form submission
  
      // Send the form data to the server using AJAX
      var xhr = new XMLHttpRequest();
      xhr.open('POST', '/signup');
      xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
      xhr.onload = function() {
        if (xhr.status === 200) {
          // Handle the server response here
          document.querySelector('#register-form').reset(); // clear the form after successful registration
          checkLoggedIn(); // check if user is logged in after registration
        } else {
          // Handle any errors here
        }
      };
      xhr.send(new FormData(this));
    });
  
    // Add event listener to the logout form
    document.getElementById('logout-form').addEventListener('submit', function(e) {
      e.preventDefault(); // prevent the default form submission
  
      // Send the logout request to the server using AJAX
      var xhr = new XMLHttpRequest();
      xhr.open('POST', '/logout');
      xhr.onload = function() {
        if (xhr.status === 200) {
          // Handle the server response here
          checkLoggedIn(); // check if user is logged in after logout
        } else {
          // Handle any errors here
        }
      };
      xhr.send();
    });
  
    // Add event listener to the register-submit-button
    document.getElementById('register-submit-button').addEventListener('click', function(e) {
      e.preventDefault(); // prevent the default form submission
  
      // Send the form data to the server using AJAX
      var xhr = new XMLHttpRequest();
      xhr.open('POST', '/signup');
      xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
      xhr.onload = function() {
        if (xhr.status === 200) {
          // Handle the server response here
          document.querySelector('.register-form').reset(); // clear the form after successful registration
          checkLoggedIn(); // check if user is logged in after registration
        } else {
          // Handle any errors here
        }
      };
      xhr.send(new FormData(document.querySelector('.register-form')));
    });
  
  });
  
  // Function to check if the user is logged in
  function checkLoggedIn() {
// Send a request
// to the server to check if the user is logged in
  var xhr = new XMLHttpRequest();
  xhr.open('GET', '/check_login');
  xhr.onload = function() {
  if (xhr.status === 200) {
  var response = JSON.parse(xhr.responseText);
  if (response.logged_in) {
  // If the user is logged in, show the necessary elements
  document.querySelector('.left-sidebar').style.display = 'block';
  document.querySelector('.right-sidebar').style.display = 'block';
  document.querySelector('.posts').style.display = 'block';
  document.querySelector('.container').style.display = 'block';
  document.getElementById('login-form').style.display = 'none';
  document.querySelector('.register-form').style.display = 'none';
  document.getElementById('logged-in-message').style.display = 'block';
  document.getElementById('logout-form').style.display = 'block';
  } else {
  // If the user is not logged in, hide the necessary elements
  document.querySelector('.left-sidebar').style.display = 'none';
  document.querySelector('.right-sidebar').style.display = 'none';
  document.querySelector('.posts').style.display = 'none';
  document.querySelector('.container').style.display = 'none';
  document.getElementById('login-form').style.display = 'block';
  document.querySelector('.register-form').style.display = 'none';
  document.getElementById('logged-in-message').style.display = 'none';
  document.getElementById('logout-form').style.display = 'none';
  }
  } else {
  // Handle any errors here
  }
  };
  xhr.send();
  }