<?php

include __DIR__ . '/vendor/autoload.php';

use GuzzleHttp\Client;
use GuzzleHttp\Psr7\Request;
use GuzzleHttp\Exception\RequestException;
use Monolog\Logger;
use Monolog\Handler\StreamHandler;

$config = require(__DIR__ . "/config.php");

$httpClient = new Client();
$logger = new Logger('sms_client');
$logger->pushHandler(new StreamHandler('php://stdout'));

if ($_SERVER['REQUEST_URI'] != '/mo' && $_SERVER['REQUEST_URI'] != '/dlr' ){
    echo "/mo and /dlr urls available only";
    exit;
}

$data = json_decode(file_get_contents('php://input'), true);

//DLR handler
if ($_SERVER['REQUEST_URI'] == '/dlr'){    
    header('Content-Type: application/json;charset=utf-8');
    $logger->info('Received DLR', $data);
    echo json_encode([
        'status' => 'success',
    ]);
    exit;
}
//mo handler
$logger->info('Received MO', $data);
ob_start();
header('Content-Type: application/json;charset=utf-8');
echo json_encode([
    'status' => 'success',
]);
ob_end_flush();
ob_flush();
flush();

//wait a bit before sending MT
sleep(1);

//send MT
$mtParams = array_merge($data, $config['mt']);
$mtParams['to'] = $mtParams['from'];
unset($mtParams['from']);
$request = new Request('POST', $config['mt_url'], ['Content-Type' => 'application/json'], json_encode($mtParams));
$logger->debug("Sending new MT to [{$config['mt_url']}] with data", $mtParams);

try{
    $response = $httpClient->send($request);
} catch (RequestException $ex) {            
    $logger->error("MT error: " . $ex->getMessage());
    exit;
}                

$responseData = json_decode($response->getBody(), true);
$logger->debug("MT response: ", $responseData);

