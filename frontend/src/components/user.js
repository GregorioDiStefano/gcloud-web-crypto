import React from 'react'
import {Fieldset, Field, createValue} from 'react-forms'
import Progress from 'react-progressbar'
import superagent from 'superagent'
import SweetAlert from 'sweetalert-react';
import Header from './header.js';
import Search from './search.js';
import {Bar} from 'react-chartjs-2';
import Humanize from 'humanize-plus'

class User extends React.Component {

  constructor(props) {
    super(props)
    let formValue = createValue({
      value: props.value
    })
    this.state = { "users":[], "stats": {}}
  }

  componentDidMount = () => {
	   this.getUsers()
	    this.getStats()
  }

  componentWillUnmount = () => {
	this.setState( { users: [], stats: {} })
  }

  getUsers = () => {
	  let self = this
	  let tempUsers = []
	  superagent
        .get('/auth/account/users')
        .end(function(error, response) {
			for (let user of response.body) {
				tempUsers.push(user)
				self.setState( { users: tempUsers })
			}
        })
  }

  getStats = () => {
	  let self = this
	  let UserStats = []
	  superagent
        .get('/auth/account/stat')
        .end(function(error, response) {
			self.setState( { stats: response.body["stats"] })
        })

  }

  handleCheckBoxChange = (event, username) => {
	  let self = this

	  if (event.target.checked) {
		  superagent
			.put('/auth/account/enable/' + username)
			.end(function(error, response) {
				for (let user of self.state.users) {
					self.getUsers()
					}
			})
      } else {
		  superagent
			.delete('/auth/account/enable/' + username)
			.end(function(error, response) {
				for (let user of self.state.users) {
					self.getUsers()
				}
			})
	  }
  }


  render() {
      const hasAccountInfo = this.state.users.length
	    const hasStats = this.state.stats.length

	    const stats = this.state.stats
	    const filesData = {
	        labels: ['0-500MB', '500MB-1GB', '1GB-2GB', '2GB-3GB', '3GB-4GB', '4GB-5GB', "over 5GB"],
	        datasets: [
		      {
		         label: 'Files',
		         backgroundColor: 'rgba(255,99,132,0.2)',
		         borderColor: 'rgba(255,10,12,1)',
		         borderWidth: .5,
		         hoverBackgroundColor: 'rgba(255,99,132,0.4)',
		         hoverBorderColor: 'rgba(255,99,132,1)',
		         data: [stats["files_0mb_500mb"],
				            stats["files_500mb_1gb"],
				            stats["files_1gb_2gb"],
				            stats["files_2gb_3gb"],
				            stats["files_3gb_4gb"],
				            stats["files_4gb_5gb"],
				            stats["files_over_5gb"]]

		      }]
	    };

	    const transmissionData = {
	        labels: ['7 days ago', '14 days ago', '30 days ago', '60 days ago', '90 days ago'],
	        datasets: [
		      {
		         label: 'Uploads',
		         backgroundColor: 'rgba(74,204,241,0.2)',
		         borderColor: 'rgba(75,99,132,1)',
		         borderWidth: .5,
		         hoverBackgroundColor: 'rgba(255,99,132,0.4)',
		         hoverBorderColor: 'rgba(255,99,132,1)',
		         data: [stats["uploads_last_7_days"],
				            stats["uploads_last_14_days"],
				            stats["uploads_last_30_days"],
				            stats["uploads_last_60_days"],
				            stats["uploads_last_90_days"]]
		      }]
	    };


    return (
    <div className="container">
      <Header/>
      <div className="flex-container">
      { hasAccountInfo > 0 &&
          <div id="enabledUsers">
              <h4> Enabled accounts </h4>
              <div className="col-lg-6 row">
	          {
				   this.state.users.map((user) => {
			           return (
			               <div className="input-group" id="enabled-accounts">
				              <span className="input-group-addon">
					               <input type="checkbox" checked={user.Enabled || user.Admin} disabled={user.Admin} onChange={(e) => this.handleCheckBoxChange(e, user.Username)}></input>
				              </span>
				           <input type="text" className="form-control" value={user.Username} readOnly disabled></input>
			               </div>
                       );
		           })
              }
          </div>
          </div>
      }



		<div style={{ width: 300, height: 300 }}>
      <h4 className="center"> Uploads </h4>
		  <Bar
			   data={transmissionData}
			   options={{
				       maintainAspectRatio: false,
			   }}
		  />
		</div>


		<div style={{ width: 300, height: 300 }}>
      <h4 className="center"> File sizes </h4>
		  <Bar
			   data={filesData}
			   options={{
				       maintainAspectRatio: false,
			   }}
		  />

		</div>

    <div id="detailFileStats">
      <dl>
        <dt>Total files</dt>
          <dd>- { this.state.stats["total_files"]}</dd>
        <dt>Total disk-space usage</dt>
          <dd>- { Humanize.fileSize(parseInt(this.state.stats["total_usage"])) }</dd>
        <dt>Total downloads </dt>
          <dd>- { this.state.stats["total_downloads"] }</dd>
      </dl>
    </div>

    </div>
    </div>
    )
  }
}

export default User
