export function addPost() {
  var title = document.getElementById('post-title').value;
  var content = document.getElementById('post-content').value;
  var selCat = Array.from(document.querySelectorAll('input[name="categories[]"]:checked')).map(function(category) {
    return {
      id: parseInt(category.getAttribute('title')),
      name: category.getAttribute('value')
    };
  });

  const uname = localStorage.getItem('username');
  const userID = parseInt(localStorage.getItem('id'));

  var postData = {
    title: title,
    content: content,
    userID: userID,
    categories: selCat,
    author: uname
  };

  console.log("PostData: ",postData)

  fetch('/addPost', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(postData)
  })
  .then(function(response) {
    if (response.ok) {
      console.log('Post data sent successfully!');
      document.getElementById('msg').innerHTML = "Your post was submitted."
      // Add any additional logic or UI updates here after successful submission
      
        // Uncheck all checkboxes after adding a post
      document.querySelectorAll('input[type="checkbox"]:checked').forEach(checkbox => checkbox.checked = false);

       // Clear all form fields after adding a post
       document.getElementById('add-post-form').reset();

      return
    } else {
      console.error('Error sending post data.');
      // Handle the error condition appropriately
      return
    }
  })
  .catch(function(error) {
    console.error('Error sending post data:', error);
    // Handle the error condition appropriately
    return
  });
};