import { showMessage, setButtonLoading } from "./notify.js";
import { showPostsView } from "./navigation.js";

export function addPost() {
  var title = document.getElementById('post-title').value;
  var content = document.getElementById('post-content').value;
  var selCat = Array.from(document.querySelectorAll('input[name="categories[]"]:checked')).map(function(category) {
    return {
      id: parseInt(category.getAttribute('title')),
      name: category.getAttribute('value')
    };
  });

  var postData = {
    title: title,
    content: content,
    categories: selCat
  };

  const submitButton = document.getElementById('add-post-submit');
  setButtonLoading(submitButton, true, 'Posting...');

  fetch('/addPost', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(postData)
  })
  .then(function(response) {
    if (response.ok) {
        // Uncheck all checkboxes after adding a post
      document.querySelectorAll('input[type="checkbox"]:checked').forEach(checkbox => checkbox.checked = false);

       // Clear all form fields after adding a post
       document.getElementById('add-post-form').reset();

      // showPostsView() clears #msg as part of resetting the view, so the
      // success message must be shown after it, not before.
      showPostsView();
      showMessage('Your post was submitted.', 'success');

      return;
    }
    return response.text().then((message) => {
      throw new Error(message || `Failed to submit post (${response.status})`);
    });
  })
  .catch(function(error) {
    showMessage("Err: " + error.message, "error");
    console.error('Error sending post data:', error);
  })
  .finally(function() {
    setButtonLoading(submitButton, false);
  });
}