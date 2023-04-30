export function createMainHTML() {
    const mainDiv = document.getElementById("main");
  
    // Navgation header
    const header = document.createElement("header");
    header.classList.add("header");
  
    const h1 = document.createElement("h1");
    h1.setAttribute("id", "title");
  
    const a = document.createElement("a");
    a.textContent = "theDialectic";
    h1.appendChild(a);
  
    const registerButton = document.createElement("button");
    registerButton.setAttribute("type", "submit");
    registerButton.classList.add("header-btns");
    registerButton.setAttribute("id", "register-button");
    registerButton.textContent = "Register";
  
    const loginButton = document.createElement("button");
    loginButton.setAttribute("type", "submit");
    loginButton.classList.add("header-btns");
    loginButton.setAttribute("id", "login-button");
    loginButton.textContent = "Login";
  
    header.appendChild(h1);
    header.appendChild(registerButton);
    header.appendChild(loginButton);
  
    mainDiv.appendChild(header);
  
    // Message div
    const messageDiv = document.createElement("div");
    messageDiv.setAttribute("id", "msg");
    mainDiv.appendChild(messageDiv);
  
    // Introductory remarks
    const introDiv = document.createElement("div");
    introDiv.classList.add("intro");
    introDiv.setAttribute("id", "intro");
  
    const h2 = document.createElement("h2");
    h2.textContent = "Welcome to theDialectic";
  
    const p = document.createElement("p");
    p.innerHTML = "Please feel free to bombard us with your conversation.<br><br>If you are not yet a member, please oblige us and register.<br>If you already have an account, login and converse.<br><br>";
  
    introDiv.appendChild(h2);
    introDiv.appendChild(p);
  
    mainDiv.appendChild(introDiv);
  
    // Login/Register Form
    const loginForm = document.createElement("form");
    loginForm.classList.add("login-form");
    loginForm.setAttribute("id", "login-form");
    loginForm.setAttribute("method", "post");
    loginForm.style.display = "none";
  
    const loginLabel = document.createElement("label");
    loginLabel.setAttribute("for", "username");
    loginLabel.textContent = "Login";
  
    const loginInput = document.createElement("input");
    loginInput.setAttribute("type", "text");
    loginInput.setAttribute("name", "username");
    loginInput.setAttribute("id", "username");
    loginInput.setAttribute("placeholder", "Enter your login");
    loginInput.required = true;
  
    const passwordLabel = document.createElement("label");
    passwordLabel.setAttribute("for", "password");
    passwordLabel.textContent = "Password";
  
    const passwordInput = document.createElement("input");
    passwordInput.setAttribute("type", "password");
    passwordInput.setAttribute("name", "password");
    passwordInput.setAttribute("id", "password");
    passwordInput.setAttribute("placeholder", "Enter your password");
    passwordInput.required = true;
  
    const loginSubmitButton = document.createElement("button");
    loginSubmitButton.setAttribute("type", "submit");
    loginSubmitButton.setAttribute("id", "login-button");
    loginSubmitButton.textContent = "Login";
  
    const registerSwitchButton = document.createElement("button");
    registerSwitchButton.setAttribute("type", "button");
    registerSwitchButton.setAttribute("id", "register-switch-button");
    registerSwitchButton.textContent = "Go to registration";
  
    loginForm.appendChild(loginLabel);
    loginForm.appendChild(loginInput);
    loginForm.appendChild(passwordLabel);
    loginForm.appendChild(passwordInput);
    loginForm.appendChild(loginSubmitButton);
    loginForm.appendChild(registerSwitchButton);
  
    //???????
    mainDiv.appendChild(loginForm);
  
  // Registration Form
  const registrationForm = document.createElement("form");
  registrationForm.classList.add("register-form");
  registrationForm.setAttribute("id", "registration-form");
  registrationForm.setAttribute("method", "post");
  registrationForm.style.display = "none";
  
  const firstnameLabel = document.createElement("label");
  firstnameLabel.setAttribute("for", "firstname");
  firstnameLabel.textContent = "First Name: ";
  
  const firstnameInput = document.createElement("input");
  firstnameInput.setAttribute("type", "text");
  firstnameInput.setAttribute("name", "firstname");
  firstnameInput.setAttribute("id", "regfname");
  firstnameInput.setAttribute("placeholder", "First Name");
  firstnameInput.required = true;
  
  const lastnameLabel = document.createElement("label");
  lastnameLabel.setAttribute("for", "lastname");
  lastnameLabel.textContent = "Last Name: ";
  
  const lastnameInput = document.createElement("input");
  lastnameInput.setAttribute("type", "text");
  lastnameInput.setAttribute("name", "lastname");
  lastnameInput.setAttribute("id", "reglname");
  lastnameInput.setAttribute("placeholder", "Last Name");
  lastnameInput.required = true;
  
  const usernameLabel = document.createElement("label");
  usernameLabel.setAttribute("for", "username");
  usernameLabel.textContent = "Username: ";
  
  const usernameInput = document.createElement("input");
  usernameInput.setAttribute("type", "text");
  usernameInput.setAttribute("name", "username");
  usernameInput.setAttribute("id", "reguname");
  usernameInput.setAttribute("placeholder", "Username");
  usernameInput.required = true;
  
  const emailLabel = document.createElement("label");
  emailLabel.setAttribute("for", "email");
  emailLabel.textContent = "Email: ";
  
  const emailInput = document.createElement("input");
  emailInput.setAttribute("type", "email");
  emailInput.setAttribute("name", "email");
  emailInput.setAttribute("id", "regemail");
  emailInput.setAttribute("placeholder", "Email");
  emailInput.required = true;
  
  const ageLabel = document.createElement("label");
  ageLabel.setAttribute("for", "age");
  ageLabel.textContent = "Age: ";
  
  const ageInput = document.createElement("input");
  ageInput.setAttribute("type", "number");
  ageInput.setAttribute("name", "age");
  ageInput.setAttribute("id", "regage");
  ageInput.setAttribute("placeholder", "Age");
  ageInput.setAttribute("min", "0");
  ageInput.setAttribute("max", "150");
  ageInput.required = true;
  
  const genderLabel = document.createElement("label");
  genderLabel.setAttribute("for", "gender");
  genderLabel.textContent = "Gender: ";
  
  const genderSelect = document.createElement("select");
  genderSelect.setAttribute("name", "gender");
  genderSelect.setAttribute("id", "reggender");
  
  const maleOption = document.createElement("option");
  maleOption.setAttribute("value", "male");
  maleOption.textContent = "Male";
  
  const femaleOption = document.createElement("option");
  femaleOption.setAttribute("value", "female");
  femaleOption.textContent = "Female";
  
  const otherOption = document.createElement("option");
  otherOption.setAttribute("value", "other");
  otherOption.textContent = "Other";
  
  genderSelect.appendChild(maleOption);
  genderSelect.appendChild(femaleOption);
  genderSelect.appendChild(otherOption);
  
  const passwordLabel2 = document.createElement("label");
  passwordLabel2.setAttribute("for", "password");
  passwordLabel2.textContent = "Password: ";
  
  const passwordInput2 = document.createElement("input");
  passwordInput2.setAttribute("type", "password");
  passwordInput2.setAttribute("name", "password");
  passwordInput2.setAttribute("id", "regpassword");
  passwordInput2.setAttribute("placeholder", "Password");
  passwordInput2.required = true;
  
  const confpasswordLabel = document.createElement("label");
  confpasswordLabel.setAttribute("for", "confpassword");
  confpasswordLabel.textContent = "Confirm Password: ";
  
  const confpasswordInput = document.createElement("input");
  confpasswordInput.setAttribute("type", "password");
  confpasswordInput.setAttribute("name", "confpassword");
  confpasswordInput.setAttribute("id", "regconfpassword");
  confpasswordInput.setAttribute("placeholder", "Confirm Password");
  confpasswordInput.required = true;
  
  const registerSubmitButton = document.createElement("button");
  registerSubmitButton.setAttribute("type", "submit");
  registerSubmitButton.setAttribute("id", "register-submit-button");
  registerSubmitButton.textContent = "Register";
  
  const loginSwitchButton = document.createElement("button");
  loginSwitchButton.setAttribute("type", "submit");
  loginSwitchButton.setAttribute("id", "login-switch-button");
  loginSwitchButton.textContent = "Go to login";
  
  registrationForm.appendChild(firstnameLabel);
  registrationForm.appendChild(firstnameInput);
  registrationForm.appendChild(lastnameLabel);
  registrationForm.appendChild(lastnameInput);
  registrationForm.appendChild(usernameLabel);
  registrationForm.appendChild(usernameInput);
  registrationForm.appendChild(emailLabel);
  registrationForm.appendChild(emailInput);
  registrationForm.appendChild(ageLabel);
  registrationForm.appendChild(ageInput);
  registrationForm.appendChild(genderLabel);
  registrationForm.appendChild(genderSelect);
  registrationForm.appendChild(passwordLabel2);
  registrationForm.appendChild(passwordInput2);
  registrationForm.appendChild(confpasswordLabel);
  registrationForm.appendChild(confpasswordInput);
  registrationForm.appendChild(registerSubmitButton);
  registrationForm.appendChild(loginSwitchButton);
  
  mainDiv.appendChild(registrationForm);
}