let form  = document.getElementById('fileSelector');
let file_elem = document.getElementById('selectedFile');
let result = document.getElementById('result')
form.addEventListener('submit', async (event) => {
    
    var fileList = file_elem.files;

    event.preventDefault();
    let formData = new FormData();
     
    formData.append("file", fileList[0]);
    const response = await fetch("/hash", {
	method: 'POST',
	body: fileList[0]
    });
    var text = await response.text();
    result.textContent = text;
});
