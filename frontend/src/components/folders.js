import React, { Component } from 'react'
import request from 'superagent'
import { BootstrapTable, TableHeaderColumn } from 'react-bootstrap-table'
import SweetAlert from 'sweetalert-react';
import FileInfo from './fileinfo.js'
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
    //this.hideFileInfoHandler = this.hideFileInfoHandler.bind(this)

    this.currentFolder = "/"
    this.updateFolder("/")
}

hideFileInfoHandler = () => {
  this.setState({ showFileInfo: false })
}

updateFolder = (path) => {
  this.setState({ showFileInfo: false})
  var self = this
  request
    .get('/auth/list/fs/?path=' + path)
    .end(function(err, res){
      if(err) throw err;
      self.setState({ fs: JSON.parse(res.text) });
    });
}

navigate = (row) => {
  if (row["type"] == "folder") {
    this.currentFolder = row.fullpath
    this.updateFolder(this.currentFolder)
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
  .del('/auth/file/' + deleteID)
  .end(function(err, res){
    console.log(err, res)
    if (err) throw err;
    self.updateFolder(self.currentFolder)
  });
}

removeFolder = (deleteID) => {
  var self = this
  request
  .del('/auth/folder?path=' + deleteID)
  .end(function(err, res){
    console.log(err, res)
    if (err) throw err;
    self.updateFolder(self.currentFolder)
  });
}

downloadFile = (e, d) => {
  e.stopPropagation()
  e.preventDefault()

  var uuid = d.ID
  request
  .get('/auth/file/' + uuid)
  .end(function(err, res){
    if (err) throw err;
  });
}

iconFormatter = (cell, row) => {
  if (row["type"] == "folder") {
    var folderDownloadLink = "/auth/folder?path=" + "/" + row.fullpath + "/"
    return (
      <div>
        <i className="glyphicon glyphicon-folder-open"></i>
        <i className="glyphicon glyphicon-remove" onClick={(evt) => this.promptRemoveDialog(evt, "folder", row.fullpath)}></i>
        <a className="glyphicon glyphicon-download-alt downloadLink" href={folderDownloadLink}></a>
      </div>
      );
  } else {
    var downloadLink = "/auth/file/" + row.id
    return (
      <div>
        <i className="glyphicon glyphicon-file" onClick={(evt) => this.displayInformation(evt, row)} ></i>
        <i className="glyphicon glyphicon-remove" onClick={(evt) => this.promptRemoveDialog(evt, "file", row.id)}></i>
        <a className="glyphicon glyphicon-download-alt downloadLink" href={downloadLink}></a>
      </div>
    );
  }
}

displayInformation = (e, row) => {
  this.setState({ fileType: row["filetype"],
                  fileDescription: row["description"],
                  fileName: row["name"],
                  fileSize: row["filesize"],
                  fileMD5Hash: row["md5"],
                  fileTags: row["tags"],
                  fileUploadDate: row["upload_date"]})
  this.setState({ showFileInfo: true })
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
        folders.push(obj)
      }
    }

    const options = {
      onRowClick: this.navigate,
      defaultSortName: 'name',
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
            <FileInfo hideFileInfoHandler = {this.hideFileInfoHandler}
                    fileName={this.state.fileName}
                    fileTitle={this.state.fileTitle}
                    fileDescription={this.state.fileDescription}
                    fileType={this.state.fileType}
                    fileMD5Hash={this.state.fileMD5Hash}
                    fileSize={this.state.fileSize}
                    fileTags={this.state.fileTags}
                    fileUploadDate={this.state.fileUploadDate}>
            </FileInfo>
        : null }

        <BootstrapTable data={folders} striped={false} hover={true} options={options} bordered={ false } condensed>
          <TableHeaderColumn isKey={true} dataField="type" dataFormat={this.iconFormatter}  dataSort width='80'></TableHeaderColumn>
          <TableHeaderColumn dataField="name" >Name</TableHeaderColumn>
          <TableHeaderColumn dataField="filesize" dataFormat={this.fileSizeFormatter} width='100'>Size</TableHeaderColumn>
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
