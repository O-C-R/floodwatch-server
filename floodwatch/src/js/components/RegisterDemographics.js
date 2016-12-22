// @flow

import React, {Component} from 'react';
import {withRouter} from 'react-router';
import {FWApiClient} from '../api/api';
import history from '../common/history';
import {ProfileExplanation, DemographicContainer} from './Profile';

type State = {
  saving: boolean;
};

function initialState(): State {
  return {
    saving: false
  };
}

export class RegisterDemographics extends Component {
  state: State;

  constructor() {
    super();
    this.state = initialState();
  }

  handleSuccess() {
    history.push('/compare');
  }

  render() {
    return (
      <div className="profile-page panel">
        <div className="panel-container">
          <h1>Your profile</h1>
          <ProfileExplanation />
        </div>
        <DemographicContainer onSuccess={this.handleSuccess.bind(this)} />
      </div>
    );
  }
}