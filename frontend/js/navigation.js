import { getAllPosts } from "./getAllPosts.js";
import { createCategoryFilter, clearActiveCategory } from "./categoryFilter.js";
import { generateCategoryDropdown } from "./generateCategories.js";
import { addPost } from "./addPost.js";

// Highlights exactly one item in the sidebar nav (All Posts / New Post /
// the active category), clearing any previous highlight. Pass null to just
// clear (used for views — search, a single post — that don't correspond to
// one specific nav destination).
export function setActiveNav(activeId) {
    document.querySelectorAll('.nav-item').forEach((el) => el.classList.remove('active'));
    if (!activeId) {
        return;
    }
    const el = document.getElementById(activeId);
    if (el) {
        el.classList.add('active');
    }
}

// Switches the logged-in view to the (paginated, unfiltered) posts list.
// Shared by the "All Posts" nav item, the site title, the "Clear search"
// button, and a successful post submission, so they all reset to the same
// consistent state.
export function showPostsView() {
    document.getElementById('msg').textContent = "";
    document.getElementById('search-input').value = "";
    document.getElementById('clear-search-button').style.display = "none";
    document.getElementById('main-content').innerHTML = "";
    clearActiveCategory();
    setActiveNav('all-posts-button');
    createCategoryFilter();
    getAllPosts();
}

// Renders the "New Post" form into the content pane. The form itself is
// static markup (no user-controlled interpolation), so an innerHTML
// template is fine here — same approach mainHTML.js/loggedInHTML.js already
// use for their own static shells.
export function showAddPostView() {
    document.getElementById('msg').textContent = "";
    document.getElementById('clear-search-button').style.display = "none";
    setActiveNav('create-post-button');

    const mainContent = document.getElementById('main-content');
    mainContent.innerHTML = `
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
    `;

    generateCategoryDropdown();

    document.getElementById('add-post-form').addEventListener('submit', function (event) {
        event.preventDefault();
        addPost();
    });
}
