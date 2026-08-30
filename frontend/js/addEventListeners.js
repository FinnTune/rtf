import { logout } from "./logout.js";
import { searchPosts } from "./search.js";
import { showPostsView, showAddPostView } from "./navigation.js";

export function addEventListeners() {
    // Add event listeners to the buttons
    document.getElementById('all-posts-button').addEventListener('click', function(event) {
        event.preventDefault();
        showPostsView();
    });

    document.getElementById('create-post-button').addEventListener('click', function(event) {
        event.preventDefault();
        showAddPostView();
    });

    document.getElementById('logout-button').addEventListener('click', function(event) {
    event.preventDefault();
    logout();
    });

    document.getElementById('title').addEventListener('click', function() {
        showPostsView();
    });

    document.getElementById('search-form').addEventListener('submit', function(event) {
        event.preventDefault();
        document.getElementById('msg').textContent = "";
        searchPosts(document.getElementById('search-input').value);
    });

    document.getElementById('clear-search-button').addEventListener('click', function() {
        showPostsView();
    });
}
