import http from 'k6/http';
	
	export const options = {
  		discardResponseBodies: true,
  		scenarios: {
   			contacts: {
      				executor: 'constant-vus',
      				vus: 10,
      				duration: '5m',
    },
  },
};
	
	export default function () {
	  http.get('https://google.com');

	}