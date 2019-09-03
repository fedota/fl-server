from keras.models import load_model

def federated_averaging(updates, model_path, ckpt_path):
    
    total_num_batches = 0
    
    # Load model architecture
    average_weight_updates = load_model(model_path)
    sum_weight_updates = load_model(model_path)
    device_weight_updates = load_model(model_path)
    
    # Calculate sum of weight updates from all devices
    for device_index in range(len(updates)):
        
        n, weight_updates_path = updates[device_index]
        
        total_num_batches += n
        
        if device_index == 0:
             sum_weight_updates.load_weights(weight_updates_path)
                
        else:
            # Load device weight updates checkpoint
            device_weight_updates.load_weights(weight_updates_path)
            
            # Add weight updates from device to prefix sum
            for layer_index in range(len(sum_weight_updates.layers)):

                # Old sum of weight updates
                old_sum_weight_updates_values = sum_weight_updates.layers[layer_index].get_weights()

                # Device weight updates      
                device_weight_updates_values = device_weight_updates.layers[layer_index].get_weights()

                # Weight updates calculation
                sum_weight_updates.layers[layer_index].set_weights(np.asarray(old_sum_weight_updates_values) 
                                                                 + np.asarray(device_weight_updates_values))

#                print("old weights: ",  old_layer_weights)
#                print("new weights: ",  new_layer_weights)
#                print("update weights: ",  update_weights.layers[i].get_weights())

    
    # Calculate average
    for layer_index in range(len(sum_weight_updates.layers)):

        # Value of sum of weight updates
        sum_weight_updates_values = sum_weight_updates.layers[layer_index].get_weights()

        # Calculate average and store
        average_weight_updates.layers[layer_index].set_weights( np.asarray(sum_weight_updates_values)/total_num_batches )

#        print("old weights: ",  old_layer_weights)
#        print("new weights: ",  new_layer_weights)
#        print("update weights: ",  update_weights.layers[i].get_weights())
            
    # Add average of weight updates to checkpoint
    # Load model and checkpoints
    model = load_model(model_path)
    model.load_weights(ckpt_path)
    
    for layer_index in range(len(model.layers)):
        
        # Average of weight updates values
        average_weight_updates_values = average_weight_updates.layers[layer_index].get_weights()
        
        old_model_values = model.layers[layer_index].get_weights()
        
        model.layers[layer_index].set_weights(np.asarray(old_model_values) + np.asarray(average_weight_updates_values))
    
    # Save updated model checkpoints
    model.save_weights(ckpt_path)