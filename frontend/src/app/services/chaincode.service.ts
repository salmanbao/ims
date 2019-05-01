import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';

@Injectable({
  providedIn: 'root'
})
export class ChaincodeService{

  constructor(private http: HttpClient) { }

  add(info: any) {
    console.log("Service Data:",info);
    return this.http.post('/api/users', info);
  }
}