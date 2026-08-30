import { addEventListeners } from "./addEventListeners.js";
import { createCategoryFilter, getPostsByCategory } from "./categoryFilter.js";
export function createLoggedInHTML() {
    const mainDiv = document.getElementById("main");
    mainDiv.innerHTML = `
    <!-- Top bar: brand, search, current user -->
    <header class="topbar">
      <h1 id="title"><a>theDialectic</a></h1>
      <form id="search-form" class="topbar-search">
        <input type="text" id="search-input" name="q" placeholder="Search posts..." maxlength="100">
        <button type="submit" class="btns" id="search-submit-button">Search</button>
        <button type="button" class="btns" id="clear-search-button" style="display: none;">Clear</button>
      </form>
      <div class="topbar-user">
        <span id="topbar-username"></span>
        <button type="submit" class="header-btns" id="logout-button">Logout</button>
      </div>
    </header>

    <div id="msg"></div>

    <div class="app-shell">
      <!-- Primary navigation: feed, post creation, category browsing -->
      <nav class="sidebar-left">
        <button type="button" class="nav-item" id="all-posts-button">All Posts</button>
        <button type="button" class="nav-item" id="create-post-button">New Post</button>
        <h4 class="sidebar-heading">Categories</h4>
        <div id="category-selection" class="category-nav"></div>
      </nav>

      <!-- Content pane: feed / single post / new-post form / search results
           all render in here (see navigation.js, getAllPosts.js, search.js) -->
      <main class="content-pane" id="main-content"></main>

      <aside class="sidebar-right">
        <div id="users">
          <h3>Users</h3>
          <ul id="users-list"></ul>
        </div>
      </aside>
    </div>
      `;

    const usernameLabel = document.getElementById('topbar-username');
    if (usernameLabel) {
        usernameLabel.textContent = localStorage.getItem('username') || '';
    }

    createCategoryFilter();
    getPostsByCategory();
    addEventListeners();
}
