import { sendMessage } from "./chat.js";
import { logout } from "./logout.js";
import { addPostHTML } from "./addPostHTML.js";

export function createLoggedInHTML() {
    const mainDiv = document.getElementById("main");
    mainDiv.innerHTML = `
    <!-- Navgation header -->
    <header class="header">
      <h1 id="title"><a>theDialectic</a></h1>
      <button type="submit" class="header-btns" id="all-posts-button">Posts</button>
      <button type="submit" class="header-btns" id="create-post-button">New Post</button>
      <button type="submit" class="header-btns" id="logout-button">Logout</button>
    </header>

    <div id="msg"></div>

    <!--Introductory remarks-->
    <div class="intro" id="intro">
      <h2>Welcome to theDialectic</h2>
      <p>
        Please feel free to bombard us with your conversation.<br><br>
        Create posts and/or comment on others.<br><br>
        Feel free to start a conversation with the chat.<br><br>
      </p>
    </div>

    <!-- Main Content -->
    <div class="main-content" id="main-content">
          <h3>Latest Posts</h3>
    </div>

    <!-- Chat -->
    <div class="chat" id ="chat">
      <h3>Chat</h3> 
          <div name="chat-messages" id="chat-messages"></div>
      <div class="chat-footer">
        <!-- If using a from tag in an SPA (single page application) you must suppress the form reloading the whole page. -->
        <form>
          <textarea type="text" id="new-message" name="new-message" placeholder="Type your message"></textarea>
          <button id="message-submit" class="btns" type="submit">Send</button>
        </form>
      </div>
      <div id ="users">
        <h3>Users</h3>
        <ul id="users-list"></ul>
      </div>
    </div>
      `;

    // Add event listeners to the buttons
    // This onsubmit uses the form to reload the page and the websocket gets reloaded which is why you cant see the message
    // document.getElementById('new-message').onsubmit = sendMessage;
    document.getElementById('chat').addEventListener('submit', function(event) {
      event.preventDefault();
      sendMessage();
    });
    document.getElementById('create-post-button').addEventListener('click', function(event) {
      event.preventDefault();
      addPostHTML();
    });
    document.getElementById('logout-button').addEventListener('click', function(event) {
      event.preventDefault();
      logout();
    });
    document.getElementById('title').addEventListener('click', function() {
      window.location.href = '/';
    });
  }