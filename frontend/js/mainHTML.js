import { login } from "./login.js";
import { register } from "./register.js";

export function createMainHTML() {
    const mainDiv = document.getElementById("main");
    mainDiv.innerHTML = `
    <!-- Navgation header -->
    <header class="header">
      <h1 id="title"><a>theDialectic</a></h1>
      <button type="submit" class="header-btns" id="register-button">Register</button>
      <button type="submit" class="header-btns" id="login-button">Login</button>
    </header>

    <div id="msg">
    </div>

    <!--Introductory remarks-->
    <div class="intro" id="intro">
      <h2>Welcome to theDialectic</h2>
      <p>
        Please feel free to bombard us with your conversation.<br><br>
        If you are not yet a member, please oblige us and register.<br>
        If you already have an account, login and converse.<br><br>
      </p>
    </div>

    <!-- Login/Register Form -->
    <form class="login-form" id="login-form" method="post" style="display: none">
      <label for="username">Login</label>
      <input type="text" name="username" id="username" placeholder="Enter your login" required>
    
      <label for="password">Password</label>
      <input type="password" name="password" id="password" placeholder="Enter your password" required>
    
      <button type="submit" id="login-button">Login</button>
      <button type="button" id="register-switch-button">Go to registration</button>
    </form>

    <!-- Registration Form -->
    <form class="register-form" id="registration-form" method="post" style="display: none;">
      <label for="firstname">First Name: </label>
      <input type="text" name="firstname" id="regfname" placeholder="First Name" required><br>
  
      <label for="lastname">Last Name: </label>
      <input type="text" name="lastname" id="reglname" placeholder="Last Name" required><br>
  
      <label for="username">Username: </label>
      <input type="text" name="username" id="reguname" placeholder="Username" required><br>
  
      <label for="email">Email: </label>
      <input type="email" name="email" id="regemail"placeholder="Email" required><br>
  
      <label for="age">Age: </label>
      <input type="number" name="age" id="regage" placeholder="Age" min="0" max="150" required><br>
  
      <label for="gender">Gender: </label>
      <select name="gender" id="reggender">
          <option value="male">Male</option>
          <option value="female">Female</option>
          <option value="other">Other</option>
      </select><br>
  
      <label for="password">Password: </label>
      <input type="password" name="password" id="regpassword" placeholder="Password" required><br>
  
      <label for="confpassword">Confirm Password: </label>
      <input type="password" name="confpassword" id="regconfpassword" placeholder="Confirm Password" required><br>
  
      <button type="submit" id="register-submit-button">Register</button>
      <button type="submit" id="login-switch-button">Go to login</button>
    </form>
    `;

    // Event listeners
    document.getElementById('title').addEventListener('click', function() {
      document.getElementById('login-form').style.display = 'none';
      document.getElementById('intro').style.display = 'flex';
      document.getElementById('registration-form').style.display = 'none';
    });

    document.getElementById('register-button').addEventListener('click', function() {
      document.getElementById('login-form').style.display = 'none';
      document.getElementById('intro').style.display = 'none';
      document.getElementById('registration-form').style.display = 'block';
    });

    document.getElementById('login-button').addEventListener('click', function() {
      document.getElementById('login-form').style.display = 'block';
      document.getElementById('intro').style.display = 'none';
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

    document.getElementById('login-form').addEventListener('submit', function(event) {
      event.preventDefault();
      login();
    });

    document.getElementById('registration-form').addEventListener('submit', function(event) {
      event.preventDefault();
      register();
    });
}