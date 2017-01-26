import React, { Component } from 'react'
import request from 'superagent'
import { BootstrapTable, TableHeaderColumn } from 'react-bootstrap-table'
import SweetAlert from 'sweetalert-react';
import FileInfo from './fileinfo'
import moment from 'moment'
import Header from './header.js';
import Search from './search.js';


class Folder extends Component {
  constructor(props) {
    super(props)

    this.state = {
      showDeleteFile: false,
      showDeleteFolder: false,
      showFileInfo: false,
    }

    this.currentFolder = "/"
    this.updateFolder("/")
}

updateFolder = (path) => {
  this.setState({ showFileInfo: false})
  var self = this
  request
    .get('http://localhost:3000/list/?path=' + path)
    .end(function(err, res){
      if(err) throw err;
      self.setState({ fs: JSON.parse(res.text) });
    });
}

navigate = (row) => {
  if (row["type"] == "folder") {
    this.currentFolder = row.fullpath
    this.updateFolder(this.currentFolder)
  } else {
    this.setState({ fileType: row["filetype"],
                    fileDescription: row["description"],
                    fileName: row["filename"],
                    fileSize: row["filesize"],
                    fileMD5Hash: row["md5"],
                    fileUploadDate: row["upload_date"]})
    this.setState({ showFileInfo: true })
  }
}

promptRemoveDialog = (e, deleteType, deleteInfo) => {
  e.stopPropagation()
  if (deleteType == "folder") {
    this.setState({ showDeleteFolder: true, deleteID: deleteInfo, deleteType: "folder" })
  } else {
    this.setState({ showDeleteFile: true, deleteID: deleteInfo, deleteType: "file" })
  }
}

removeFile = (deleteID) => {
  var self = this
  request
  .del('http://127.0.0.1:3000/file/' + deleteID)
  .end(function(err, res){
    console.log(err, res)
    if (err) throw err;
    self.updateFolder(self.currentFolder)
  });
}

removeFolder = (deleteID) => {
  var self = this
  request
  .del('http://127.0.0.1:3000/folder?path=' + deleteID)
  .end(function(err, res){
    console.log(err, res)
    if (err) throw err;
    self.updateFolder(self.currentFolder)
  });
}

downloadFile = (e, d) => {
  e.stopPropagation()
  var uuid = d.ID
  request
  .get('http://127.0.0.1:3000/file/' + uuid)
  .end(function(err, res){
    if (err) throw err;
  });
}

iconFormatter = (cell, row) => {
  if (row["type"] == "folder") {
    var folderDownloadLink = "http://127.0.0.1:3000/folder?path=" + "/" + row.fullpath + "/"
    return (
      <div>
        <i className="glyphicon glyphicon-folder-open"></i>
        <i className="glyphicon glyphicon-remove" onClick={(evt) => this.promptRemoveDialog(evt, "folder", row.fullpath)}></i>
        <a className="glyphicon glyphicon-download-alt downloadLink" href={folderDownloadLink}></a>
      </div>
      );
  } else {
    var downloadLink = "http://127.0.0.1:3000/file/" + row.id
    return (
      <div>
        <i className="glyphicon glyphicon-file" ></i>
        <i className="glyphicon glyphicon-remove" onClick={(evt) => this.promptRemoveDialog(evt, "file", row.id)}></i>
        <a className="glyphicon glyphicon-download-alt downloadLink" href={downloadLink}></a>
      </div>
    );
  }
}

displayInformation = (e) => {
  this.setState({file: "test", fi: "abc", fileid: e.id})
}

fileSizeFormatter = (cell, row) => {
  let i = -1,
      fileSizeInBytes = cell,
      byteUnits = [' kB', ' MB', ' GB', ' TB', 'PB', 'EB', 'ZB', 'YB'];

  if (fileSizeInBytes == 0 || row["type"] == "folder") {
    return ""
  }

  do {
      fileSizeInBytes = fileSizeInBytes / 1024;
      i++;
  } while (fileSizeInBytes > 1024);

  return Math.max(fileSizeInBytes, 0.05).toFixed(1) + byteUnits[i];
}

normalizeDate = (cell, row) => {
  let d = moment(cell, "YYYY-MM-DD[T]hh:mm:ss").fromNow()
  if (typeof(d) != "string" || d == "Invalid date") {
    console.log("Invalid date: ", cell)
  } else {
      return (
        <i> {d} </i>
      );
  }
}

render() {
    var sourcedata

    if (this.state && typeof this.state.fs !== 'undefined') {
      sourcedata = this.state.fs
      console.log(sourcedata)
    } else {
      console.log(sourcedata)
    }

    var folders = [];

    for (var key in sourcedata) {
      if (sourcedata.hasOwnProperty(key)) {
        var obj = sourcedata[key]
        console.dir(obj)
        folders.push(obj)


          /*{ "id": key,
                       "type": obj["Type"],
                       "folder": obj["Path"],
                       "fullpath": obj["FullPath"],
                       "location": obj["ObjectData"]["Folder"],
                       "size": obj["ObjectData"]["FileSize"],
                       "publishedDate": obj["ObjectData"]["UploadDate"],
                       "fullDetails": obj})
                       */
        console.log(obj)
      }
    }

    const options = {
      onRowClick: this.navigate,
      defaultSortName: 'folder',
      defaultSortOrder: 'asc'
    };

    let currentFolderLinks = () => {
      let parentPath = "", returnLink = []

      if (this.currentFolder == "/") return "/";

      returnLink.push(
        <a key={parentPath} onClick={()=> { this.updateFolder("/"); this.currentFolder = "/" }}> /    </a>
      )

      for (let folder of this.currentFolder.split("/")) {
        if (folder.length === 0) {
          continue
        }

        folder = folder + "/"
        let newPath = parentPath + folder

        returnLink.push(
          <a key={newPath} onClick={()=> { this.updateFolder(newPath); this.currentFolder = "/" + newPath + "/"; }}> {folder} </a>
        )

        parentPath += folder
      }

      return returnLink
    }

    console.log(currentFolderLinks())

    return (
      <div>
        <Header />

        <h2 id="locationBar">{currentFolderLinks()}</h2>
        <Search />

        { this.state.showFileInfo ?
            <FileInfo fileName={this.state.fileName}
                    fileTitle={this.state.fileTitle}
                    fileDescription={this.state.fileDescription}
                    fileType={this.state.fileType}
                    fileMD5Hash={this.state.fileMD5Hash}
                    fileSize={this.state.fileSize}
                    fileUploadDate={this.state.fileUploadDate}>
            </FileInfo>
        : null }

        <BootstrapTable data={folders} striped={false} hover={true} options={options} bordered={ false } condensed>
          <TableHeaderColumn isKey={true} dataField="type" dataFormat={this.iconFormatter} onClick={this.displayInformation} dataSort width='80'></TableHeaderColumn>
          <TableHeaderColumn dataField="folder">Name</TableHeaderColumn>
          <TableHeaderColumn dataField="size" dataFormat={this.fileSizeFormatter} width='100'>Size</TableHeaderColumn>
          <TableHeaderColumn dataField="upload_date" dataFormat={this.normalizeDate} width='150'>Uploaded</TableHeaderColumn>
        </BootstrapTable>
        <SweetAlert
          show={this.state.showDeleteFile}
          title="Delete"
          text="Are you sure you want to delete the file?"
          showCancelButton={true} onCancel={() => this.setState({ showDeleteFile: false}) }
          onConfirm={() => { this.removeFile(this.state.deleteID); this.setState({ showDeleteFile: false}); } }
          />
        <SweetAlert
          show={this.state.showDeleteFolder}
          title="Delete"
          text={"Are you sure you want to delete the folder? " + this.state.deleteID}
          showCancelButton={true} onCancel={() => this.setState({ showDeleteFolder: false}) }
          onConfirm={() => { this.removeFolder(this.state.deleteID); this.setState({ showDeleteFolder: false}); } }
          />
      </div>
    );
  }
}

export default Folder