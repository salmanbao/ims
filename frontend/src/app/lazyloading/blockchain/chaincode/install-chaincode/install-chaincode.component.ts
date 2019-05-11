import { Component, OnInit, Inject } from '@angular/core';
import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material';
import { ChaincodeService } from 'app/services/chaincode.service';
import { FormControl, Validators } from '@angular/forms';

export interface DialogData {
  title: string;
  peers: Array<string>;
  chaincodeName: string;
  chaincodeVersion: string;
  chaincodePath:string;
  chaincodeType: string;
}
 
@Component({
  selector: 'app-install-chaincode',
  templateUrl: './install-chaincode.component.html',
  styleUrls: ['./install-chaincode.component.scss']
})
export class InstallChaincodeComponent implements OnInit {
  peers = new FormControl('',Validators.required);
  languages = ['golang', 'node'];
  peersList = ["peer0.org1.example.com","peer1.org1.example.com"];
  paths:Array<string>;
  chaincodeFilesObj = {};
  constructor(
    private chaincodeService:ChaincodeService,
    public dialogRef: MatDialogRef<InstallChaincodeComponent>,
    @Inject(MAT_DIALOG_DATA) public data: DialogData) {
    data.title = 'Install chaincode'
  }

  ngOnInit() {
    this.chaincodeService.getChaincodeFiles().subscribe(
      res=>{
        this.chaincodeFilesObj = res;
        this.paths = Object.keys(res);
      }
    );
  }

  installChaincode(){
    this.data.chaincodePath = this.chaincodeFilesObj[this.data.chaincodePath];
    this.chaincodeService.installChaincode(this.data).subscribe(
      res => {console.log(res);}
    );  
  }
  onNoClick(): void {
    this.dialogRef.close();
  }

}
