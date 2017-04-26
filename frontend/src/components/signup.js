import React, { Component } from 'react'
import request from 'superagent'
import { browserHistory } from 'react-router'

class Signup extends Component {
  constructor(props) {
    super(props)

    this.state = {password1: "", password2: "", disabledSubmit: true}
  }

  onSubmit = (e) => {
    e.preventDefault()
    var self = this;
    request
      .post('/account/signup')
      .set('Content-Type', 'application/json')
      .send({ "password": self.state.password1 })
      .end(function(error, response){
        if(error) {
          if (response.statusCode == 403) {
            self.setState({"signupFail": true})
          } else if (response.statusCode == 409) {
	    self.setState({"accountExists": true})
	  }
        } else {
            // hack
            setTimeout(function(){
               browserHistory.push(`/login`)
            }, 250);
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
    return (
       <form className="form-horizontal"  onSubmit={this.onSubmit}>
         <fieldset>
           <legend>Signup</legend>
           <div className="form-group">
             <label className="col-md-4 control-label" htmlFor="login">Username</label>
             <div className="col-md-4">
               <input id="login" name="login" className="form-control input-md" value="admin" disabled readOnly/>
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
             <input type="submit" id="singlebutton" name="singlebutton" className="center btn btn-primary" value="Signup" disabled={this.state.disabledSubmit}/>
           </div>
           </div>
         </fieldset>
       </form>
     );
   }
}

export default Signup
