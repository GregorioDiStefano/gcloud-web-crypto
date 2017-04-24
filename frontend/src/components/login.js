import React, { Component } from 'react'
import request from 'superagent'
import { BootstrapTable, TableHeaderColumn } from 'react-bootstrap-table';
import { Router, Route, Redirect, Link, browserHistory } from 'react-router'

class Login extends Component {
  constructor(props) {
    super(props)

    this.state = {
      wrongPassword: false
    }
  }

  onSubmit = (e) => {
    e.preventDefault()
    var self = this;
    request
      .post('/account/login')
      .set('Content-Type', 'application/json')
      .send({ "username": "admin", "password": document.forms.item(0)[1].value })
      .end(function(error, response){
        if(error) {
          if (response.statusCode == 401) {
            self.setState({"wrongPassword": true})
          }
        } else {
          self.setState({"wrongPassword": false})

          var token = JSON.parse(response.text)["token"]
          document.cookie = "jwt=" + token;

          if (self.props.location.state != null) {
            browserHistory.push(self.props.location.state.nextPathname)
          } else {
            browserHistory.push(`/dir`)
          }

        }
      });
  }

  render () {
    return (
       <form className="form-horizontal"  onSubmit={this.onSubmit}>
         <fieldset>
           <legend>Login</legend>
           <div className="form-group">
             <label className="col-md-4 control-label" htmlFor="password">Password</label>
             <div className="col-md-4">
               <input id="password" name="password" type="password" className="form-control input-md" required />
             </div>
           </div>
           <div className="form-group">
             <label className="col-md-4 control-label" htmlFor="singlebutton" />
             <div className="col-md-4">
               { this.state.wrongPassword &&
                 <p className="center"> Incorrect password </p>
               }
               <input type="submit" id="singlebutton" name="singlebutton" className="center btn btn-primary" value="Login"/>
           </div>
           </div>
         </fieldset>
       </form>
     );
   }
}

export default Login
