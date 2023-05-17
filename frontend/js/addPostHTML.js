import { addEventListeners } from "./addEventListeners.js";
import {generateCategoryDropdown} from "./generateCategories.js";

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
            <div id="categories"></div><br>
            <button type="submit" id="add-post-submit">Submit Post</button>
        </form>
    </div>

    <!--Introductory remarks-->
    <div class="intro" id="intro">
      <h2>Welcome to yourDialectic</h2>
      <p>
        Please feel free to bombard us with your conversation.<br><br>
        Create posts and/or comment on others.<br><br>
        Feel free to start a conversation with the chat.<br><br>
      </p>
    </div>

    <!-- Category Selection-->
    <div id="category-selection"></div>

    <!-- Main Content -->
    <div class="main-content" id="main-content">
    </div>

    <!-- Chat -->
    <div id ="chat" class="chat-window">
      <h3>Chat</h3> 
          <div name="chat-messages" id="chat-messages" class="chat-messages"></div>
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
  addEventListeners();
  generateCategoryDropdown();
}
