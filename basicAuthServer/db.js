var mongoose = require('mongoose');
mongoose.connect('mongodb://mongo/db').catch(error => handleConnectionError(error));

function handleConnectionError(error){
    console.log("error connecting to mongo")
    console.error(error)
    setTimeout(function() {retryConnect()}, 5000)
}

function retryConnect(){
    console.log("second try")
    mongoose.connect('mongodb://mongo/db')
}