import { addEventListeners } from "./addEventListeners.js";
import { createCategoryFilter, getPostsByCategory } from "./categoryFilter.js";
import { generateCategoryDropdown } from "./generateCategories.js";
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
      <h2>Welcome to yourDialectic</h2>
      <p>
        Please feel free to bombard us with your conversation.<br><br>
        Create posts, comment, and chat your heart out.<br><br>
      </p>
    </div>

    <!-- Add Post -->
    <div class="add-post" id="add-post">
        <h3>Add Post</h3>
        <form id="add-post-form">
            <label for="post-title">Title:</label><br>
            <input type="text" id="post-title" name="title" maxlength="100" required><br>
            <label for="post-content">Content:</label><br>
            <textarea type="text" id="post-content" cols="50" rows="4" name="content" maxlength="2000" required></textarea><br><br>
            <div id="categories"></div><br>
            <button type="submit" id="add-post-submit">Submit Post</button>
        </form>
    </div>


    <!-- Search -->
    <form id="search-form">
      <input type="text" id="search-input" name="q" placeholder="Search posts..." maxlength="100">
      <button type="submit" class="btns" id="search-submit-button">Search</button>
      <button type="button" class="btns" id="clear-search-button" style="display: none;">Clear</button>
    </form>

    <!-- Category Selection-->
    <div id="category-selection"></div>

    <!-- Main Content -->
    <div class="main-content" id="main-content">
    </div>

    <div id ="users">
        <h3>Users</h3>
        <ul id="users-list"></ul>
      </div>
      `;  
  createCategoryFilter();
  generateCategoryDropdown();
  getPostsByCategory();
  addEventListeners();
}