let form  = document.getElementById('fileSelector');
let file_elem = document.getElementById('selectedFile');
let result = document.getElementById('result')
let progress = document.getElementById('upload-progress')
form.addEventListener('submit', async (event) => {

    var fileList = file_elem.files;

    event.preventDefault();
    let formData = new FormData();
    formData.append("file", fileList[0]);

    var req = new XMLHttpRequest();
    req.submittedData = formData;
    
    req.onload = function(e){
	result.textContent = req.response;
    }
    req.open("post", "/hash");
    function updateProgress (oEvent) {
	if (oEvent.lengthComputable) {
	    var percentComplete = oEvent.loaded / oEvent.total * 100;
	    progress.value = percentComplete;
	}
    }

    req.upload.addEventListener("progress", updateProgress);
    req.send(formData);

    
    // the fetch API does not provide a way of monitoring upload progress -_-
    /*
    const response = await fetch("/hash", {
	method: 'POST',
	body: fileList[0]
    });
    var text = await response.text();
    result.textContent = text;*/
});
