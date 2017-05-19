import React, { Component } from 'react'
import request from 'superagent'
import { browserHistory } from 'react-router'

class Signup extends Component {
  constructor(props) {
    super(props)

    this.state = {password1: "", password2: "", disabledSubmit: true, signupError: "", username: "admin", mustBeAdmin: false}
  }

  componentDidMount = () => {
    request
      .get('/account/initial')
      .end(function(error, response) {
         if (response.statusCode == 404) {
           this.setState({ "mustBeAdmin": true })
         }
      })
    }


  handleUserChange = (event) => {
    this.setState({ username: event.target.value});
  }

  onSubmit = (e) => {
    e.preventDefault()
    var self = this;
    request
      .post('/account/signup')
      .set('Content-Type', 'application/json')
      .send({ "password": self.state.password1, "username": self.state.username })
      .end(function(error, response){
        if (response.statusCode != 201) {
          if ("status" in response.body) {
            self.setState({signupError: response.body["status"]})
          }
        } else {
            // hack: redirect to login after 800ms since operation takes some time
            setTimeout(function(){
               browserHistory.push(`/login`)
            }, 800);
          }
        })
      }

  verifyPassword = (e) => {
    self = this
    this.setState({[e.target.id]:  e.target.value}, function(){
      if (this.state.password1 == this.state.password2 && this.state.password1 != "") {
        self.setState({disabledSubmit: false})
      } else {
        self.setState({disabledSubmit: true})
      }
    })
  }

  render () {
    const mustBeAdmin = this.state.mustBeAdmin;

    return (
       <form className="form-horizontal"  onSubmit={this.onSubmit}>
         <fieldset>
           <legend>Signup</legend>
           <div className="form-group">
             <label className="col-md-4 control-label" htmlFor="login">Username</label>
             <div className="col-md-4">
             { mustBeAdmin ? (
                 <input id="login" name="username" className="form-control input-md" value={this.state.username} readOnly/>
             ) : (
                 <input id="login" name="username" className="form-control input-md" onChange={this.handleUserChange} value={this.state.username} />
             )}
             </div>
           </div>
           <div className="form-group">
             <label className="col-md-4 control-label" htmlFor="password">Password</label>
             <div className="col-md-4">
               <input id="password1" name="password" type="password" className="form-control input-md" onChange={this.verifyPassword} required />
             </div>
           </div>
           <div className="form-group">
             <label className="col-md-4 control-label" htmlFor="password">Password (repeat)</label>
             <div className="col-md-4">
               <input id="password2" name="password" type="password" className="form-control input-md" onChange={this.verifyPassword} required />
             </div>
           </div>
           <div className="form-group">
             <label className="col-md-4 control-label" htmlFor="singlebutton" />
             <div className="col-md-4">
             <p className="center"> {this.state.signupError}</p>
             <input type="submit" id="singlebutton" name="singlebutton" className="center btn btn-primary" value="Signup" disabled={this.state.disabledSubmit}/>
           </div>
           </div>
         </fieldset>
       </form>
     );
   }
}

export default Signup
