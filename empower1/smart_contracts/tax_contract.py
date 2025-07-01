# Smart Contract for managing and applying taxes based on IRE inputs.

from empower1.smart_contracts.base_contract import BaseContract, SmartContractError
# from empower1.transaction import Transaction # If needed for constructing tax transactions

# Address of the IRE system or a trusted AI oracle that provides tax calculation data
IRE_TAX_ORACLE_ADDRESS = "Emp1_IRE_Oracle_TaxCalc002"
TAX_COLLECTION_WALLET = "Emp1_Treasury_TaxWallet_001" # Where collected taxes are sent

class TaxContract(BaseContract):
    """
    A smart contract to manage the application and collection of taxes
    on transactions, based on rules potentially defined or adjusted by the IRE.
    """
    def __init__(self, contract_address, owner_address, blockchain_interface=None, ire_oracle_address=IRE_TAX_ORACLE_ADDRESS):
        super().__init__(contract_address, owner_address, blockchain_interface)
        self.version = "0.1.0-tax"
        self.ire_oracle_address = ire_oracle_address
        self.tax_rules = {} # Stores tax rules, e.g., by user category or transaction type
        self.default_tax_rate_affluent = 0.09 # 9% as per README, can be overridden
        self.tax_collection_address = TAX_COLLECTION_WALLET

        # Example rule:
        self.tax_rules["affluent_user_default"] = {
            "rate": self.default_tax_rate_affluent,
            "description": "Default tax for affluent users."
        }
        self._emit_event("TaxContractDeployed", {"oracle": self.ire_oracle_address, "collection_wallet": self.tax_collection_address})
        print(f"TaxContract initialized at {contract_address}, Oracle: {self.ire_oracle_address}")

    def set_tax_rule(self, caller_address: str, rule_id: str, rate: float, description: str = ""):
        """
        Allows the owner or a designated admin (or IRE oracle itself) to set/update tax rules.
        Args:
            caller_address (str): Address making the call (must be authorized).
            rule_id (str): A unique identifier for the tax rule (e.g., "affluent_high_value_tx").
            rate (float): The tax rate (e.g., 0.05 for 5%).
            description (str, optional): Description of the rule.
        """
        if caller_address != self.owner_address and caller_address != self.ire_oracle_address:
            raise SmartContractError("Only owner or IRE oracle can set tax rules.")
        if not (0 <= rate <= 1): # Rate should be between 0% and 100%
            raise SmartContractError("Tax rate must be between 0 and 1.")

        self.tax_rules[rule_id] = {"rate": rate, "description": description}
        self._emit_event("TaxRuleSet", {"rule_id": rule_id, "rate": rate, "setter": caller_address})
        return {"status": "success", "message": f"Tax rule '{rule_id}' set to {rate*100}%."}

    def get_tax_rule(self, caller_address: str, rule_id: str):
        """Retrieves a specific tax rule."""
        if rule_id not in self.tax_rules:
            raise SmartContractError(f"Rule ID '{rule_id}' not found.")
        return self.tax_rules[rule_id]

    def calculate_and_apply_tax(self, caller_address: str, transaction_details: dict, sender_profile: dict):
        """
        Calculates tax for a given transaction and sender profile.
        This method would typically be called by the blockchain's transaction processing logic
        or by the IRE itself when a transaction occurs.

        Args:
            caller_address (str): The entity invoking this (e.g., blockchain system, IRE).
                                  Needs to be authorized if this method directly moves funds.
            transaction_details (dict): {'amount': 100, 'asset_id': 'EMP', 'sender': 'addr1', 'receiver': 'addr2'}
            sender_profile (dict): {'user_id': 'addr1', 'wealth_category': 'affluent', ...} (from IRE Oracle)

        Returns:
            dict: {'tax_amount': X, 'tax_asset_id': Y, 'tax_applied_rule_id': Z, 'collected': True/False}
                  Or generates a tax transaction to be processed by the blockchain.
        """
        # For simplicity, this simulation assumes this method is called by a trusted entity (e.g. IRE itself or blockchain hook)
        # and the "caller_address" has authority or the method just returns calculation.
        # If this method were to *directly* pull funds, caller_address would need to be the sender or an approved spender.

        # Determine applicable rule (simplified - could be complex logic from IRE oracle)
        tax_rate = 0.0
        applied_rule_id = "none"
        wealth_category = sender_profile.get("wealth_category", "unknown")

        if wealth_category == "affluent":
            # Use a specific rule if available, else default
            rule = self.tax_rules.get("affluent_user_default") # Example rule
            if rule:
                tax_rate = rule["rate"]
                applied_rule_id = "affluent_user_default"
        # More rules could be checked here based on transaction_details or sender_profile

        transaction_amount = transaction_details.get("amount", 0)
        tax_amount = transaction_amount * tax_rate
        asset_id = transaction_details.get("asset_id", "EMP") # Default to EMP if not specified

        if tax_amount > 0:
            # In a real blockchain, this step is crucial:
            # Option 1: This contract *instructs* a transfer from sender to tax wallet.
            #           Requires the contract to have approval or be part of core protocol.
            # Option 2: This contract is *called by the sender* during their transaction,
            #           and the sender's transaction includes this tax as an output.
            # Option 3: This function *returns* the tax details, and another process handles collection.

            # For simulation, let's assume Option 3, or a simplified direct transfer if called by a system process.
            # If this contract were to manage the collection itself:
            # success = self._transfer_funds(
            #     from_address=transaction_details["sender"],
            #     to_address=self.tax_collection_address,
            #     amount=tax_amount,
            #     asset_id=asset_id
            # )
            # For this placeholder, we'll just log and assume collection happens externally or is signaled.
            self._emit_event("TaxCalculated", {
                "original_sender": transaction_details["sender"],
                "transaction_amount": transaction_amount,
                "tax_amount": tax_amount,
                "asset_id": asset_id,
                "rule_id": applied_rule_id,
                "destination_wallet": self.tax_collection_address
            })
            print(f"Tax calculated: {tax_amount} {asset_id} for tx by {transaction_details['sender']}. Rule: {applied_rule_id}.")
            # This method would return the tax details for the main system to act upon.
            # The actual "collection" (transfer) would be a separate transaction or part of the original.
            return {
                "tax_amount": tax_amount,
                "tax_asset_id": asset_id,
                "tax_applied_rule_id": applied_rule_id,
                "tax_destination": self.tax_collection_address,
                "message": "Tax calculated. Collection to be handled by the transaction processing system."
            }
        else:
            return {"tax_amount": 0, "message": "No tax applicable based on current rules."}

    def update_collection_address(self, caller_address: str, new_collection_address: str):
        """Allows the owner to update the tax collection wallet address."""
        if caller_address != self.owner_address:
            raise SmartContractError("Only the owner can update the collection address.")
        self.tax_collection_address = new_collection_address
        self._emit_event("TaxCollectionAddressUpdated", {"new_address": new_collection_address})
        return {"status": "success", "message": f"Tax collection address updated to {new_collection_address}."}

    def get_contract_info(self, caller_address: str = None):
        """Returns specific information for this contract."""
        base_info = super().get_contract_info(caller_address)
        base_info.update({
            "ire_oracle_address": self.ire_oracle_address,
            "tax_collection_address": self.tax_collection_address,
            "number_of_tax_rules": len(self.tax_rules)
        })
        return base_info

if __name__ == "__main__":
    # Example Usage
    class MockBlockchainTax(BaseContract.MockBlockchain if hasattr(BaseContract, 'MockBlockchain') else object):
        def log_event(self, contract_address, event_name, event_data):
             print(f"[TaxContract Log] From {contract_address} - Event: {event_name}, Data: {event_data}")
        def get_current_timestamp(self): # Not directly used by TaxContract but good for base
            import time
            return time.time()

    mock_bc_tax = MockBlockchainTax()
    owner_addr_tax = "Owner_Tax_Contract_Deployer"
    tax_contract_addr = "SC_TaxApplicator_001"
    ire_tax_oracle = IRE_TAX_ORACLE_ADDRESS

    tax_sc = TaxContract(
        contract_address=tax_contract_addr,
        owner_address=owner_addr_tax,
        blockchain_interface=mock_bc_tax,
        ire_oracle_address=ire_tax_oracle
    )
    print(f"\n--- TaxContract ({tax_sc.version}) Info ---")
    print(tax_sc.invoke_method("get_contract_info", owner_addr_tax, {}))

    print("\n--- Setting a new Tax Rule (by Owner) ---")
    tax_sc.invoke_method("set_tax_rule", owner_addr_tax, {
        "rule_id": "low_value_exempt",
        "rate": 0.0, # 0% tax
        "description": "Exemption for low value transactions for all users."
    })
    print(tax_sc.invoke_method("get_tax_rule", owner_addr_tax, {"rule_id": "low_value_exempt"}))

    print("\n--- Calculating Tax (simulation - called by a trusted system entity) ---")
    # Affluent user, standard transaction
    tx_details_1 = {'amount': 1000, 'asset_id': 'EMP', 'sender': 'User_Alice_Affluent', 'receiver': 'User_Bob'}
    sender_profile_1 = {'user_id': 'User_Alice_Affluent', 'wealth_category': 'affluent'}
    tax_result_1 = tax_sc.invoke_method("calculate_and_apply_tax", ire_tax_oracle, { # Called by oracle
        "transaction_details": tx_details_1,
        "sender_profile": sender_profile_1
    })
    print(f"Tax Result 1: {tax_result_1}")
    # Expected tax: 1000 * 0.09 = 90 EMP

    # Non-affluent user, standard transaction
    tx_details_2 = {'amount': 500, 'asset_id': 'EMP', 'sender': 'User_Charlie_Regular', 'receiver': 'User_David'}
    sender_profile_2 = {'user_id': 'User_Charlie_Regular', 'wealth_category': 'medium'}
    tax_result_2 = tax_sc.invoke_method("calculate_and_apply_tax", ire_tax_oracle, {
        "transaction_details": tx_details_2,
        "sender_profile": sender_profile_2
    })
    print(f"Tax Result 2: {tax_result_2}")
    # Expected tax: 0 (as no rule for 'medium' is defined to apply tax)

    print("\n--- Updating Collection Address (by Owner) ---")
    new_treasury = "Emp1_Main_Treasury_Updated_002"
    tax_sc.invoke_method("update_collection_address", owner_addr_tax, {"new_collection_address": new_treasury})
    updated_info = tax_sc.invoke_method("get_contract_info", owner_addr_tax, {})
    assert updated_info["tax_collection_address"] == new_treasury
    print(f"Updated contract info: {updated_info}")

    print("\nTaxContract placeholder demo complete.")
