import { logout } from "./logout.js";
import { sendMessage } from "./chat.js";
import {addPost} from "./addPost.js";

export function addPostHTML() {
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

    <!-- Add Post -->
    <div class="add-post" id="add-post">
        <h3>Add Post</h3>
        <form id="add-post-form">
            <label for="title">Title:</label><br>
            <input type="text" id="post-title" name="title"><br>
            <label for="content">Content:</label><br>
            <textarea type="text" id="post-content" cols="50" rows="4" name="content"></textarea><br><br>
            <button type="submit" id="add-post-submit">Submit Post</button>
        </form>
    </div>

    <!-- Main Content -->
    <div class="main-content" id="main-content">
          <h3>Latest Posts</h3>
          <table id="posts-table">
            <tr>
              <th>Title</th>
              <th>Content</th>
              <th>Author</th>
              <th>Created</th>
            </tr>
          </table>
    </div>

    <!-- Chat -->
    <div class="chat" id ="chat">
      <h3>Chat</h3> 
          <div name="chat-messages" id="chat-messages"></div>
      <div class="chat-footer">
        <!-- If using a from tag in an SPA (single page application) you must suppress the form reloading the whole page. -->
        <form>
          <textarea type="text" id="new-message" name="new-message" placeholder="Type your message"></textarea>
          <button id="message-submit" type="submit">Send</button>
        </form>
      </div>
    </div>
    <div id ="users">
      <h3>Users</h3>
      <ul id="users-list"></ul>
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
    document.getElementById('add-post-submit').addEventListener('submit', function(event) {
        event.preventDefault();
        addPost();
      });
      document.getElementById('title').addEventListener('click', function() {
        window.location.href = '/';
      });
  }