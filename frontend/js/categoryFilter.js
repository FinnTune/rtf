import { createPostsTable, getAllPosts, showEmptyState, renderPostRows, renderPagination, POSTS_PAGE_SIZE } from "./getAllPosts.js";
import { clearTable } from "./getAllPosts.js";
import { showMessage } from "./notify.js";
import { getCategories } from "./categories.js";

export async function createCategoryFilter() {
    const categoryFilterDiv = document.getElementById('category-selection');
    categoryFilterDiv.className = "category-filter";
    categoryFilterDiv.replaceChildren();
    const heading = document.createElement('h4');
    heading.textContent = 'Filter by Category:';
    categoryFilterDiv.appendChild(heading);

    let categories;
    try {
        categories = await getCategories();
    } catch (error) {
        showMessage("Err: " + error.message, "error");
        console.log("Err: ", error);
        return categoryFilterDiv;
    }

    for (let category of categories) {
        const checkbox = document.createElement('input');
        checkbox.type = 'checkbox';
        checkbox.name = 'category';
        checkbox.value = category.name;
        checkbox.id = 'filter-category-' + category.id;

        const label = document.createElement('label')
        label.htmlFor = checkbox.id;
        label.appendChild(document.createTextNode(category.name));

         // Create a new div
         const div = document.createElement('div');

         // Append checkbox and label to the div
         div.appendChild(checkbox);
         div.appendChild(label);

        categoryFilterDiv.appendChild(div);

        // Add event listener. Wrapped so the change Event object doesn't
        // land in getPostsByCategory's offset parameter — a fresh filter
        // selection always starts back at the first page.
        checkbox.addEventListener('change', () => getPostsByCategory());
    }

    return categoryFilterDiv;  // Line break for readability
}

export function getPostsByCategory(offset = 0) {
    document.getElementById('main-content').innerHTML = "";
    console.log("Getting posts by category. offset=", offset)
     // Collect all the selected categories
     let selectedCategories = Array.from(document.querySelectorAll('input[type="checkbox"]:checked')).map(checkbox => checkbox.value);
    console.log("Selected categories: ", selectedCategories);
     // If no categories are selected, get all posts instead
     if (selectedCategories.length == 0) {
         return getAllPosts();
     }

     fetch(`getPostsByCategory?limit=${POSTS_PAGE_SIZE}&offset=${offset}`, {
         method: 'POST',
         mode: 'cors',
         headers: {
             'Content-Type': 'application/json'
         },
         body: JSON.stringify({ categories: selectedCategories }) // Send the selected categories to the server
     }).then((response) => {
        if(response.ok){
            console.log("Received posts by category.")
            if (document.getElementById('posts')) {
                clearTable();
            } else {
                createPostsTable();
            }
            const total = parseInt(response.headers.get('X-Total-Count') || '0', 10);
            return response.json().then((posts) => ({ posts, total }));
        }
        return response.text().then((message) => {
            throw new Error(message || `Failed to load posts (${response.status})`);
        });
    }).then(({ posts, total }) => {
        console.log("PostsAft: ", posts)
        // Posts already arrive newest-first from the server.
        if (offset > 0 && posts.length === 0) {
            // The page we asked for is now empty (e.g. the last post on it
            // was deleted) — snap back to the previous page.
            getPostsByCategory(Math.max(0, offset - POSTS_PAGE_SIZE));
            return;
        }
        let table = document.getElementById('posts-table');
        let tbody = table.querySelector('tbody');
        if (posts.length == 0) {
            showEmptyState(tbody, "No posts for this category.");
        } else {
            renderPostRows(tbody, posts);
        }
        renderPagination(offset, POSTS_PAGE_SIZE, total, getPostsByCategory);
    }).catch((error) => {
        showMessage("Err: " + error.message, "error");
        console.log("Err: ", error);
    });
    return
}