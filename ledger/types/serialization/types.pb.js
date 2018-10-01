/*eslint-disable block-scoped-var, no-redeclare, no-control-regex, no-prototype-builtins*/
(function(global, factory) { /* global define, require, module */

    /* AMD */ if (typeof define === 'function' && define.amd)
        define(["protobufjs/minimal"], factory);

    /* CommonJS */ else if (typeof require === 'function' && typeof module === 'object' && module && module.exports)
        module.exports = factory(require("protobufjs/minimal"));

})(this, function($protobuf) {
    "use strict";

    // Common aliases
    var $Reader = $protobuf.Reader, $Writer = $protobuf.Writer, $util = $protobuf.util;
    
    // Exported root namespace
    var $root = $protobuf.roots["default"] || ($protobuf.roots["default"] = {});
    
    $root.serialization = (function() {
    
        /**
         * Namespace serialization.
         * @exports serialization
         * @namespace
         */
        var serialization = {};
    
        serialization.PublicKey = (function() {
    
            /**
             * Properties of a PublicKey.
             * @memberof serialization
             * @interface IPublicKey
             * @property {Uint8Array|null} [data] PublicKey data
             */
    
            /**
             * Constructs a new PublicKey.
             * @memberof serialization
             * @classdesc Represents a PublicKey.
             * @implements IPublicKey
             * @constructor
             * @param {serialization.IPublicKey=} [properties] Properties to set
             */
            function PublicKey(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * PublicKey data.
             * @member {Uint8Array} data
             * @memberof serialization.PublicKey
             * @instance
             */
            PublicKey.prototype.data = $util.newBuffer([]);
    
            /**
             * Creates a new PublicKey instance using the specified properties.
             * @function create
             * @memberof serialization.PublicKey
             * @static
             * @param {serialization.IPublicKey=} [properties] Properties to set
             * @returns {serialization.PublicKey} PublicKey instance
             */
            PublicKey.create = function create(properties) {
                return new PublicKey(properties);
            };
    
            /**
             * Encodes the specified PublicKey message. Does not implicitly {@link serialization.PublicKey.verify|verify} messages.
             * @function encode
             * @memberof serialization.PublicKey
             * @static
             * @param {serialization.IPublicKey} message PublicKey message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            PublicKey.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.data != null && message.hasOwnProperty("data"))
                    writer.uint32(/* id 2, wireType 2 =*/18).bytes(message.data);
                return writer;
            };
    
            /**
             * Encodes the specified PublicKey message, length delimited. Does not implicitly {@link serialization.PublicKey.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.PublicKey
             * @static
             * @param {serialization.IPublicKey} message PublicKey message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            PublicKey.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes a PublicKey message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.PublicKey
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.PublicKey} PublicKey
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            PublicKey.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.PublicKey();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 2:
                        message.data = reader.bytes();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes a PublicKey message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.PublicKey
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.PublicKey} PublicKey
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            PublicKey.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies a PublicKey message.
             * @function verify
             * @memberof serialization.PublicKey
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            PublicKey.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.data != null && message.hasOwnProperty("data"))
                    if (!(message.data && typeof message.data.length === "number" || $util.isString(message.data)))
                        return "data: buffer expected";
                return null;
            };
    
            /**
             * Creates a PublicKey message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.PublicKey
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.PublicKey} PublicKey
             */
            PublicKey.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.PublicKey)
                    return object;
                var message = new $root.serialization.PublicKey();
                if (object.data != null)
                    if (typeof object.data === "string")
                        $util.base64.decode(object.data, message.data = $util.newBuffer($util.base64.length(object.data)), 0);
                    else if (object.data.length)
                        message.data = object.data;
                return message;
            };
    
            /**
             * Creates a plain object from a PublicKey message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.PublicKey
             * @static
             * @param {serialization.PublicKey} message PublicKey
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            PublicKey.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults)
                    object.data = options.bytes === String ? "" : [];
                if (message.data != null && message.hasOwnProperty("data"))
                    object.data = options.bytes === String ? $util.base64.encode(message.data, 0, message.data.length) : options.bytes === Array ? Array.prototype.slice.call(message.data) : message.data;
                return object;
            };
    
            /**
             * Converts this PublicKey to JSON.
             * @function toJSON
             * @memberof serialization.PublicKey
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            PublicKey.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return PublicKey;
        })();
    
        serialization.PrivateKey = (function() {
    
            /**
             * Properties of a PrivateKey.
             * @memberof serialization
             * @interface IPrivateKey
             * @property {Uint8Array|null} [data] PrivateKey data
             */
    
            /**
             * Constructs a new PrivateKey.
             * @memberof serialization
             * @classdesc Represents a PrivateKey.
             * @implements IPrivateKey
             * @constructor
             * @param {serialization.IPrivateKey=} [properties] Properties to set
             */
            function PrivateKey(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * PrivateKey data.
             * @member {Uint8Array} data
             * @memberof serialization.PrivateKey
             * @instance
             */
            PrivateKey.prototype.data = $util.newBuffer([]);
    
            /**
             * Creates a new PrivateKey instance using the specified properties.
             * @function create
             * @memberof serialization.PrivateKey
             * @static
             * @param {serialization.IPrivateKey=} [properties] Properties to set
             * @returns {serialization.PrivateKey} PrivateKey instance
             */
            PrivateKey.create = function create(properties) {
                return new PrivateKey(properties);
            };
    
            /**
             * Encodes the specified PrivateKey message. Does not implicitly {@link serialization.PrivateKey.verify|verify} messages.
             * @function encode
             * @memberof serialization.PrivateKey
             * @static
             * @param {serialization.IPrivateKey} message PrivateKey message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            PrivateKey.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.data != null && message.hasOwnProperty("data"))
                    writer.uint32(/* id 2, wireType 2 =*/18).bytes(message.data);
                return writer;
            };
    
            /**
             * Encodes the specified PrivateKey message, length delimited. Does not implicitly {@link serialization.PrivateKey.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.PrivateKey
             * @static
             * @param {serialization.IPrivateKey} message PrivateKey message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            PrivateKey.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes a PrivateKey message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.PrivateKey
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.PrivateKey} PrivateKey
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            PrivateKey.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.PrivateKey();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 2:
                        message.data = reader.bytes();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes a PrivateKey message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.PrivateKey
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.PrivateKey} PrivateKey
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            PrivateKey.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies a PrivateKey message.
             * @function verify
             * @memberof serialization.PrivateKey
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            PrivateKey.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.data != null && message.hasOwnProperty("data"))
                    if (!(message.data && typeof message.data.length === "number" || $util.isString(message.data)))
                        return "data: buffer expected";
                return null;
            };
    
            /**
             * Creates a PrivateKey message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.PrivateKey
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.PrivateKey} PrivateKey
             */
            PrivateKey.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.PrivateKey)
                    return object;
                var message = new $root.serialization.PrivateKey();
                if (object.data != null)
                    if (typeof object.data === "string")
                        $util.base64.decode(object.data, message.data = $util.newBuffer($util.base64.length(object.data)), 0);
                    else if (object.data.length)
                        message.data = object.data;
                return message;
            };
    
            /**
             * Creates a plain object from a PrivateKey message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.PrivateKey
             * @static
             * @param {serialization.PrivateKey} message PrivateKey
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            PrivateKey.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults)
                    object.data = options.bytes === String ? "" : [];
                if (message.data != null && message.hasOwnProperty("data"))
                    object.data = options.bytes === String ? $util.base64.encode(message.data, 0, message.data.length) : options.bytes === Array ? Array.prototype.slice.call(message.data) : message.data;
                return object;
            };
    
            /**
             * Converts this PrivateKey to JSON.
             * @function toJSON
             * @memberof serialization.PrivateKey
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            PrivateKey.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return PrivateKey;
        })();
    
        serialization.Signature = (function() {
    
            /**
             * Properties of a Signature.
             * @memberof serialization
             * @interface ISignature
             * @property {Uint8Array|null} [data] Signature data
             */
    
            /**
             * Constructs a new Signature.
             * @memberof serialization
             * @classdesc Represents a Signature.
             * @implements ISignature
             * @constructor
             * @param {serialization.ISignature=} [properties] Properties to set
             */
            function Signature(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * Signature data.
             * @member {Uint8Array} data
             * @memberof serialization.Signature
             * @instance
             */
            Signature.prototype.data = $util.newBuffer([]);
    
            /**
             * Creates a new Signature instance using the specified properties.
             * @function create
             * @memberof serialization.Signature
             * @static
             * @param {serialization.ISignature=} [properties] Properties to set
             * @returns {serialization.Signature} Signature instance
             */
            Signature.create = function create(properties) {
                return new Signature(properties);
            };
    
            /**
             * Encodes the specified Signature message. Does not implicitly {@link serialization.Signature.verify|verify} messages.
             * @function encode
             * @memberof serialization.Signature
             * @static
             * @param {serialization.ISignature} message Signature message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Signature.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.data != null && message.hasOwnProperty("data"))
                    writer.uint32(/* id 2, wireType 2 =*/18).bytes(message.data);
                return writer;
            };
    
            /**
             * Encodes the specified Signature message, length delimited. Does not implicitly {@link serialization.Signature.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.Signature
             * @static
             * @param {serialization.ISignature} message Signature message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Signature.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes a Signature message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.Signature
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.Signature} Signature
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Signature.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.Signature();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 2:
                        message.data = reader.bytes();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes a Signature message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.Signature
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.Signature} Signature
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Signature.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies a Signature message.
             * @function verify
             * @memberof serialization.Signature
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            Signature.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.data != null && message.hasOwnProperty("data"))
                    if (!(message.data && typeof message.data.length === "number" || $util.isString(message.data)))
                        return "data: buffer expected";
                return null;
            };
    
            /**
             * Creates a Signature message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.Signature
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.Signature} Signature
             */
            Signature.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.Signature)
                    return object;
                var message = new $root.serialization.Signature();
                if (object.data != null)
                    if (typeof object.data === "string")
                        $util.base64.decode(object.data, message.data = $util.newBuffer($util.base64.length(object.data)), 0);
                    else if (object.data.length)
                        message.data = object.data;
                return message;
            };
    
            /**
             * Creates a plain object from a Signature message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.Signature
             * @static
             * @param {serialization.Signature} message Signature
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Signature.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults)
                    object.data = options.bytes === String ? "" : [];
                if (message.data != null && message.hasOwnProperty("data"))
                    object.data = options.bytes === String ? $util.base64.encode(message.data, 0, message.data.length) : options.bytes === Array ? Array.prototype.slice.call(message.data) : message.data;
                return object;
            };
    
            /**
             * Converts this Signature to JSON.
             * @function toJSON
             * @memberof serialization.Signature
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Signature.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return Signature;
        })();
    
        serialization.Validator = (function() {
    
            /**
             * Properties of a Validator.
             * @memberof serialization
             * @interface IValidator
             * @property {Uint8Array|null} [pubKey] Validator pubKey
             * @property {number|Long|null} [stake] Validator stake
             */
    
            /**
             * Constructs a new Validator.
             * @memberof serialization
             * @classdesc Represents a Validator.
             * @implements IValidator
             * @constructor
             * @param {serialization.IValidator=} [properties] Properties to set
             */
            function Validator(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * Validator pubKey.
             * @member {Uint8Array} pubKey
             * @memberof serialization.Validator
             * @instance
             */
            Validator.prototype.pubKey = $util.newBuffer([]);
    
            /**
             * Validator stake.
             * @member {number|Long} stake
             * @memberof serialization.Validator
             * @instance
             */
            Validator.prototype.stake = $util.Long ? $util.Long.fromBits(0,0,false) : 0;
    
            /**
             * Creates a new Validator instance using the specified properties.
             * @function create
             * @memberof serialization.Validator
             * @static
             * @param {serialization.IValidator=} [properties] Properties to set
             * @returns {serialization.Validator} Validator instance
             */
            Validator.create = function create(properties) {
                return new Validator(properties);
            };
    
            /**
             * Encodes the specified Validator message. Does not implicitly {@link serialization.Validator.verify|verify} messages.
             * @function encode
             * @memberof serialization.Validator
             * @static
             * @param {serialization.IValidator} message Validator message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Validator.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.pubKey != null && message.hasOwnProperty("pubKey"))
                    writer.uint32(/* id 1, wireType 2 =*/10).bytes(message.pubKey);
                if (message.stake != null && message.hasOwnProperty("stake"))
                    writer.uint32(/* id 2, wireType 0 =*/16).int64(message.stake);
                return writer;
            };
    
            /**
             * Encodes the specified Validator message, length delimited. Does not implicitly {@link serialization.Validator.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.Validator
             * @static
             * @param {serialization.IValidator} message Validator message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Validator.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes a Validator message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.Validator
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.Validator} Validator
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Validator.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.Validator();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.pubKey = reader.bytes();
                        break;
                    case 2:
                        message.stake = reader.int64();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes a Validator message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.Validator
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.Validator} Validator
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Validator.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies a Validator message.
             * @function verify
             * @memberof serialization.Validator
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            Validator.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.pubKey != null && message.hasOwnProperty("pubKey"))
                    if (!(message.pubKey && typeof message.pubKey.length === "number" || $util.isString(message.pubKey)))
                        return "pubKey: buffer expected";
                if (message.stake != null && message.hasOwnProperty("stake"))
                    if (!$util.isInteger(message.stake) && !(message.stake && $util.isInteger(message.stake.low) && $util.isInteger(message.stake.high)))
                        return "stake: integer|Long expected";
                return null;
            };
    
            /**
             * Creates a Validator message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.Validator
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.Validator} Validator
             */
            Validator.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.Validator)
                    return object;
                var message = new $root.serialization.Validator();
                if (object.pubKey != null)
                    if (typeof object.pubKey === "string")
                        $util.base64.decode(object.pubKey, message.pubKey = $util.newBuffer($util.base64.length(object.pubKey)), 0);
                    else if (object.pubKey.length)
                        message.pubKey = object.pubKey;
                if (object.stake != null)
                    if ($util.Long)
                        (message.stake = $util.Long.fromValue(object.stake)).unsigned = false;
                    else if (typeof object.stake === "string")
                        message.stake = parseInt(object.stake, 10);
                    else if (typeof object.stake === "number")
                        message.stake = object.stake;
                    else if (typeof object.stake === "object")
                        message.stake = new $util.LongBits(object.stake.low >>> 0, object.stake.high >>> 0).toNumber();
                return message;
            };
    
            /**
             * Creates a plain object from a Validator message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.Validator
             * @static
             * @param {serialization.Validator} message Validator
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Validator.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults) {
                    object.pubKey = options.bytes === String ? "" : [];
                    if ($util.Long) {
                        var long = new $util.Long(0, 0, false);
                        object.stake = options.longs === String ? long.toString() : options.longs === Number ? long.toNumber() : long;
                    } else
                        object.stake = options.longs === String ? "0" : 0;
                }
                if (message.pubKey != null && message.hasOwnProperty("pubKey"))
                    object.pubKey = options.bytes === String ? $util.base64.encode(message.pubKey, 0, message.pubKey.length) : options.bytes === Array ? Array.prototype.slice.call(message.pubKey) : message.pubKey;
                if (message.stake != null && message.hasOwnProperty("stake"))
                    if (typeof message.stake === "number")
                        object.stake = options.longs === String ? String(message.stake) : message.stake;
                    else
                        object.stake = options.longs === String ? $util.Long.prototype.toString.call(message.stake) : options.longs === Number ? new $util.LongBits(message.stake.low >>> 0, message.stake.high >>> 0).toNumber() : message.stake;
                return object;
            };
    
            /**
             * Converts this Validator to JSON.
             * @function toJSON
             * @memberof serialization.Validator
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Validator.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return Validator;
        })();
    
        serialization.OverspendingProof = (function() {
    
            /**
             * Properties of an OverspendingProof.
             * @memberof serialization
             * @interface IOverspendingProof
             * @property {number|Long|null} [reserveSequence] OverspendingProof reserveSequence
             * @property {Array.<serialization.IServicePaymentTx>|null} [servicePayments] OverspendingProof servicePayments
             */
    
            /**
             * Constructs a new OverspendingProof.
             * @memberof serialization
             * @classdesc Represents an OverspendingProof.
             * @implements IOverspendingProof
             * @constructor
             * @param {serialization.IOverspendingProof=} [properties] Properties to set
             */
            function OverspendingProof(properties) {
                this.servicePayments = [];
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * OverspendingProof reserveSequence.
             * @member {number|Long} reserveSequence
             * @memberof serialization.OverspendingProof
             * @instance
             */
            OverspendingProof.prototype.reserveSequence = $util.Long ? $util.Long.fromBits(0,0,false) : 0;
    
            /**
             * OverspendingProof servicePayments.
             * @member {Array.<serialization.IServicePaymentTx>} servicePayments
             * @memberof serialization.OverspendingProof
             * @instance
             */
            OverspendingProof.prototype.servicePayments = $util.emptyArray;
    
            /**
             * Creates a new OverspendingProof instance using the specified properties.
             * @function create
             * @memberof serialization.OverspendingProof
             * @static
             * @param {serialization.IOverspendingProof=} [properties] Properties to set
             * @returns {serialization.OverspendingProof} OverspendingProof instance
             */
            OverspendingProof.create = function create(properties) {
                return new OverspendingProof(properties);
            };
    
            /**
             * Encodes the specified OverspendingProof message. Does not implicitly {@link serialization.OverspendingProof.verify|verify} messages.
             * @function encode
             * @memberof serialization.OverspendingProof
             * @static
             * @param {serialization.IOverspendingProof} message OverspendingProof message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            OverspendingProof.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.reserveSequence != null && message.hasOwnProperty("reserveSequence"))
                    writer.uint32(/* id 1, wireType 0 =*/8).int64(message.reserveSequence);
                if (message.servicePayments != null && message.servicePayments.length)
                    for (var i = 0; i < message.servicePayments.length; ++i)
                        $root.serialization.ServicePaymentTx.encode(message.servicePayments[i], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
                return writer;
            };
    
            /**
             * Encodes the specified OverspendingProof message, length delimited. Does not implicitly {@link serialization.OverspendingProof.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.OverspendingProof
             * @static
             * @param {serialization.IOverspendingProof} message OverspendingProof message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            OverspendingProof.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes an OverspendingProof message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.OverspendingProof
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.OverspendingProof} OverspendingProof
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            OverspendingProof.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.OverspendingProof();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.reserveSequence = reader.int64();
                        break;
                    case 2:
                        if (!(message.servicePayments && message.servicePayments.length))
                            message.servicePayments = [];
                        message.servicePayments.push($root.serialization.ServicePaymentTx.decode(reader, reader.uint32()));
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes an OverspendingProof message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.OverspendingProof
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.OverspendingProof} OverspendingProof
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            OverspendingProof.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies an OverspendingProof message.
             * @function verify
             * @memberof serialization.OverspendingProof
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            OverspendingProof.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.reserveSequence != null && message.hasOwnProperty("reserveSequence"))
                    if (!$util.isInteger(message.reserveSequence) && !(message.reserveSequence && $util.isInteger(message.reserveSequence.low) && $util.isInteger(message.reserveSequence.high)))
                        return "reserveSequence: integer|Long expected";
                if (message.servicePayments != null && message.hasOwnProperty("servicePayments")) {
                    if (!Array.isArray(message.servicePayments))
                        return "servicePayments: array expected";
                    for (var i = 0; i < message.servicePayments.length; ++i) {
                        var error = $root.serialization.ServicePaymentTx.verify(message.servicePayments[i]);
                        if (error)
                            return "servicePayments." + error;
                    }
                }
                return null;
            };
    
            /**
             * Creates an OverspendingProof message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.OverspendingProof
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.OverspendingProof} OverspendingProof
             */
            OverspendingProof.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.OverspendingProof)
                    return object;
                var message = new $root.serialization.OverspendingProof();
                if (object.reserveSequence != null)
                    if ($util.Long)
                        (message.reserveSequence = $util.Long.fromValue(object.reserveSequence)).unsigned = false;
                    else if (typeof object.reserveSequence === "string")
                        message.reserveSequence = parseInt(object.reserveSequence, 10);
                    else if (typeof object.reserveSequence === "number")
                        message.reserveSequence = object.reserveSequence;
                    else if (typeof object.reserveSequence === "object")
                        message.reserveSequence = new $util.LongBits(object.reserveSequence.low >>> 0, object.reserveSequence.high >>> 0).toNumber();
                if (object.servicePayments) {
                    if (!Array.isArray(object.servicePayments))
                        throw TypeError(".serialization.OverspendingProof.servicePayments: array expected");
                    message.servicePayments = [];
                    for (var i = 0; i < object.servicePayments.length; ++i) {
                        if (typeof object.servicePayments[i] !== "object")
                            throw TypeError(".serialization.OverspendingProof.servicePayments: object expected");
                        message.servicePayments[i] = $root.serialization.ServicePaymentTx.fromObject(object.servicePayments[i]);
                    }
                }
                return message;
            };
    
            /**
             * Creates a plain object from an OverspendingProof message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.OverspendingProof
             * @static
             * @param {serialization.OverspendingProof} message OverspendingProof
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            OverspendingProof.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.arrays || options.defaults)
                    object.servicePayments = [];
                if (options.defaults)
                    if ($util.Long) {
                        var long = new $util.Long(0, 0, false);
                        object.reserveSequence = options.longs === String ? long.toString() : options.longs === Number ? long.toNumber() : long;
                    } else
                        object.reserveSequence = options.longs === String ? "0" : 0;
                if (message.reserveSequence != null && message.hasOwnProperty("reserveSequence"))
                    if (typeof message.reserveSequence === "number")
                        object.reserveSequence = options.longs === String ? String(message.reserveSequence) : message.reserveSequence;
                    else
                        object.reserveSequence = options.longs === String ? $util.Long.prototype.toString.call(message.reserveSequence) : options.longs === Number ? new $util.LongBits(message.reserveSequence.low >>> 0, message.reserveSequence.high >>> 0).toNumber() : message.reserveSequence;
                if (message.servicePayments && message.servicePayments.length) {
                    object.servicePayments = [];
                    for (var j = 0; j < message.servicePayments.length; ++j)
                        object.servicePayments[j] = $root.serialization.ServicePaymentTx.toObject(message.servicePayments[j], options);
                }
                return object;
            };
    
            /**
             * Converts this OverspendingProof to JSON.
             * @function toJSON
             * @memberof serialization.OverspendingProof
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            OverspendingProof.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return OverspendingProof;
        })();
    
        serialization.Coin = (function() {
    
            /**
             * Properties of a Coin.
             * @memberof serialization
             * @interface ICoin
             * @property {string|null} [denom] Coin denom
             * @property {number|Long|null} [amount] Coin amount
             */
    
            /**
             * Constructs a new Coin.
             * @memberof serialization
             * @classdesc Represents a Coin.
             * @implements ICoin
             * @constructor
             * @param {serialization.ICoin=} [properties] Properties to set
             */
            function Coin(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * Coin denom.
             * @member {string} denom
             * @memberof serialization.Coin
             * @instance
             */
            Coin.prototype.denom = "";
    
            /**
             * Coin amount.
             * @member {number|Long} amount
             * @memberof serialization.Coin
             * @instance
             */
            Coin.prototype.amount = $util.Long ? $util.Long.fromBits(0,0,false) : 0;
    
            /**
             * Creates a new Coin instance using the specified properties.
             * @function create
             * @memberof serialization.Coin
             * @static
             * @param {serialization.ICoin=} [properties] Properties to set
             * @returns {serialization.Coin} Coin instance
             */
            Coin.create = function create(properties) {
                return new Coin(properties);
            };
    
            /**
             * Encodes the specified Coin message. Does not implicitly {@link serialization.Coin.verify|verify} messages.
             * @function encode
             * @memberof serialization.Coin
             * @static
             * @param {serialization.ICoin} message Coin message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Coin.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.denom != null && message.hasOwnProperty("denom"))
                    writer.uint32(/* id 1, wireType 2 =*/10).string(message.denom);
                if (message.amount != null && message.hasOwnProperty("amount"))
                    writer.uint32(/* id 2, wireType 0 =*/16).int64(message.amount);
                return writer;
            };
    
            /**
             * Encodes the specified Coin message, length delimited. Does not implicitly {@link serialization.Coin.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.Coin
             * @static
             * @param {serialization.ICoin} message Coin message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Coin.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes a Coin message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.Coin
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.Coin} Coin
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Coin.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.Coin();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.denom = reader.string();
                        break;
                    case 2:
                        message.amount = reader.int64();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes a Coin message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.Coin
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.Coin} Coin
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Coin.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies a Coin message.
             * @function verify
             * @memberof serialization.Coin
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            Coin.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.denom != null && message.hasOwnProperty("denom"))
                    if (!$util.isString(message.denom))
                        return "denom: string expected";
                if (message.amount != null && message.hasOwnProperty("amount"))
                    if (!$util.isInteger(message.amount) && !(message.amount && $util.isInteger(message.amount.low) && $util.isInteger(message.amount.high)))
                        return "amount: integer|Long expected";
                return null;
            };
    
            /**
             * Creates a Coin message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.Coin
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.Coin} Coin
             */
            Coin.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.Coin)
                    return object;
                var message = new $root.serialization.Coin();
                if (object.denom != null)
                    message.denom = String(object.denom);
                if (object.amount != null)
                    if ($util.Long)
                        (message.amount = $util.Long.fromValue(object.amount)).unsigned = false;
                    else if (typeof object.amount === "string")
                        message.amount = parseInt(object.amount, 10);
                    else if (typeof object.amount === "number")
                        message.amount = object.amount;
                    else if (typeof object.amount === "object")
                        message.amount = new $util.LongBits(object.amount.low >>> 0, object.amount.high >>> 0).toNumber();
                return message;
            };
    
            /**
             * Creates a plain object from a Coin message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.Coin
             * @static
             * @param {serialization.Coin} message Coin
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Coin.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults) {
                    object.denom = "";
                    if ($util.Long) {
                        var long = new $util.Long(0, 0, false);
                        object.amount = options.longs === String ? long.toString() : options.longs === Number ? long.toNumber() : long;
                    } else
                        object.amount = options.longs === String ? "0" : 0;
                }
                if (message.denom != null && message.hasOwnProperty("denom"))
                    object.denom = message.denom;
                if (message.amount != null && message.hasOwnProperty("amount"))
                    if (typeof message.amount === "number")
                        object.amount = options.longs === String ? String(message.amount) : message.amount;
                    else
                        object.amount = options.longs === String ? $util.Long.prototype.toString.call(message.amount) : options.longs === Number ? new $util.LongBits(message.amount.low >>> 0, message.amount.high >>> 0).toNumber() : message.amount;
                return object;
            };
    
            /**
             * Converts this Coin to JSON.
             * @function toJSON
             * @memberof serialization.Coin
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Coin.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return Coin;
        })();
    
        serialization.ReservedFund = (function() {
    
            /**
             * Properties of a ReservedFund.
             * @memberof serialization
             * @interface IReservedFund
             * @property {Array.<serialization.ICoin>|null} [collateral] ReservedFund collateral
             * @property {Array.<serialization.ICoin>|null} [initialFund] ReservedFund initialFund
             * @property {Array.<serialization.ICoin>|null} [usedFund] ReservedFund usedFund
             * @property {Array.<Uint8Array>|null} [resourceIDs] ReservedFund resourceIDs
             * @property {number|null} [endBlockHeight] ReservedFund endBlockHeight
             * @property {number|null} [reserveSequence] ReservedFund reserveSequence
             * @property {Array.<serialization.ITransferRecord>|null} [transferRecord] ReservedFund transferRecord
             */
    
            /**
             * Constructs a new ReservedFund.
             * @memberof serialization
             * @classdesc Represents a ReservedFund.
             * @implements IReservedFund
             * @constructor
             * @param {serialization.IReservedFund=} [properties] Properties to set
             */
            function ReservedFund(properties) {
                this.collateral = [];
                this.initialFund = [];
                this.usedFund = [];
                this.resourceIDs = [];
                this.transferRecord = [];
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * ReservedFund collateral.
             * @member {Array.<serialization.ICoin>} collateral
             * @memberof serialization.ReservedFund
             * @instance
             */
            ReservedFund.prototype.collateral = $util.emptyArray;
    
            /**
             * ReservedFund initialFund.
             * @member {Array.<serialization.ICoin>} initialFund
             * @memberof serialization.ReservedFund
             * @instance
             */
            ReservedFund.prototype.initialFund = $util.emptyArray;
    
            /**
             * ReservedFund usedFund.
             * @member {Array.<serialization.ICoin>} usedFund
             * @memberof serialization.ReservedFund
             * @instance
             */
            ReservedFund.prototype.usedFund = $util.emptyArray;
    
            /**
             * ReservedFund resourceIDs.
             * @member {Array.<Uint8Array>} resourceIDs
             * @memberof serialization.ReservedFund
             * @instance
             */
            ReservedFund.prototype.resourceIDs = $util.emptyArray;
    
            /**
             * ReservedFund endBlockHeight.
             * @member {number} endBlockHeight
             * @memberof serialization.ReservedFund
             * @instance
             */
            ReservedFund.prototype.endBlockHeight = 0;
    
            /**
             * ReservedFund reserveSequence.
             * @member {number} reserveSequence
             * @memberof serialization.ReservedFund
             * @instance
             */
            ReservedFund.prototype.reserveSequence = 0;
    
            /**
             * ReservedFund transferRecord.
             * @member {Array.<serialization.ITransferRecord>} transferRecord
             * @memberof serialization.ReservedFund
             * @instance
             */
            ReservedFund.prototype.transferRecord = $util.emptyArray;
    
            /**
             * Creates a new ReservedFund instance using the specified properties.
             * @function create
             * @memberof serialization.ReservedFund
             * @static
             * @param {serialization.IReservedFund=} [properties] Properties to set
             * @returns {serialization.ReservedFund} ReservedFund instance
             */
            ReservedFund.create = function create(properties) {
                return new ReservedFund(properties);
            };
    
            /**
             * Encodes the specified ReservedFund message. Does not implicitly {@link serialization.ReservedFund.verify|verify} messages.
             * @function encode
             * @memberof serialization.ReservedFund
             * @static
             * @param {serialization.IReservedFund} message ReservedFund message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ReservedFund.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.collateral != null && message.collateral.length)
                    for (var i = 0; i < message.collateral.length; ++i)
                        $root.serialization.Coin.encode(message.collateral[i], writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                if (message.initialFund != null && message.initialFund.length)
                    for (var i = 0; i < message.initialFund.length; ++i)
                        $root.serialization.Coin.encode(message.initialFund[i], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
                if (message.usedFund != null && message.usedFund.length)
                    for (var i = 0; i < message.usedFund.length; ++i)
                        $root.serialization.Coin.encode(message.usedFund[i], writer.uint32(/* id 3, wireType 2 =*/26).fork()).ldelim();
                if (message.resourceIDs != null && message.resourceIDs.length)
                    for (var i = 0; i < message.resourceIDs.length; ++i)
                        writer.uint32(/* id 4, wireType 2 =*/34).bytes(message.resourceIDs[i]);
                if (message.endBlockHeight != null && message.hasOwnProperty("endBlockHeight"))
                    writer.uint32(/* id 5, wireType 0 =*/40).int32(message.endBlockHeight);
                if (message.reserveSequence != null && message.hasOwnProperty("reserveSequence"))
                    writer.uint32(/* id 6, wireType 0 =*/48).int32(message.reserveSequence);
                if (message.transferRecord != null && message.transferRecord.length)
                    for (var i = 0; i < message.transferRecord.length; ++i)
                        $root.serialization.TransferRecord.encode(message.transferRecord[i], writer.uint32(/* id 7, wireType 2 =*/58).fork()).ldelim();
                return writer;
            };
    
            /**
             * Encodes the specified ReservedFund message, length delimited. Does not implicitly {@link serialization.ReservedFund.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.ReservedFund
             * @static
             * @param {serialization.IReservedFund} message ReservedFund message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ReservedFund.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes a ReservedFund message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.ReservedFund
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.ReservedFund} ReservedFund
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ReservedFund.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.ReservedFund();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        if (!(message.collateral && message.collateral.length))
                            message.collateral = [];
                        message.collateral.push($root.serialization.Coin.decode(reader, reader.uint32()));
                        break;
                    case 2:
                        if (!(message.initialFund && message.initialFund.length))
                            message.initialFund = [];
                        message.initialFund.push($root.serialization.Coin.decode(reader, reader.uint32()));
                        break;
                    case 3:
                        if (!(message.usedFund && message.usedFund.length))
                            message.usedFund = [];
                        message.usedFund.push($root.serialization.Coin.decode(reader, reader.uint32()));
                        break;
                    case 4:
                        if (!(message.resourceIDs && message.resourceIDs.length))
                            message.resourceIDs = [];
                        message.resourceIDs.push(reader.bytes());
                        break;
                    case 5:
                        message.endBlockHeight = reader.int32();
                        break;
                    case 6:
                        message.reserveSequence = reader.int32();
                        break;
                    case 7:
                        if (!(message.transferRecord && message.transferRecord.length))
                            message.transferRecord = [];
                        message.transferRecord.push($root.serialization.TransferRecord.decode(reader, reader.uint32()));
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes a ReservedFund message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.ReservedFund
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.ReservedFund} ReservedFund
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ReservedFund.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies a ReservedFund message.
             * @function verify
             * @memberof serialization.ReservedFund
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ReservedFund.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.collateral != null && message.hasOwnProperty("collateral")) {
                    if (!Array.isArray(message.collateral))
                        return "collateral: array expected";
                    for (var i = 0; i < message.collateral.length; ++i) {
                        var error = $root.serialization.Coin.verify(message.collateral[i]);
                        if (error)
                            return "collateral." + error;
                    }
                }
                if (message.initialFund != null && message.hasOwnProperty("initialFund")) {
                    if (!Array.isArray(message.initialFund))
                        return "initialFund: array expected";
                    for (var i = 0; i < message.initialFund.length; ++i) {
                        var error = $root.serialization.Coin.verify(message.initialFund[i]);
                        if (error)
                            return "initialFund." + error;
                    }
                }
                if (message.usedFund != null && message.hasOwnProperty("usedFund")) {
                    if (!Array.isArray(message.usedFund))
                        return "usedFund: array expected";
                    for (var i = 0; i < message.usedFund.length; ++i) {
                        var error = $root.serialization.Coin.verify(message.usedFund[i]);
                        if (error)
                            return "usedFund." + error;
                    }
                }
                if (message.resourceIDs != null && message.hasOwnProperty("resourceIDs")) {
                    if (!Array.isArray(message.resourceIDs))
                        return "resourceIDs: array expected";
                    for (var i = 0; i < message.resourceIDs.length; ++i)
                        if (!(message.resourceIDs[i] && typeof message.resourceIDs[i].length === "number" || $util.isString(message.resourceIDs[i])))
                            return "resourceIDs: buffer[] expected";
                }
                if (message.endBlockHeight != null && message.hasOwnProperty("endBlockHeight"))
                    if (!$util.isInteger(message.endBlockHeight))
                        return "endBlockHeight: integer expected";
                if (message.reserveSequence != null && message.hasOwnProperty("reserveSequence"))
                    if (!$util.isInteger(message.reserveSequence))
                        return "reserveSequence: integer expected";
                if (message.transferRecord != null && message.hasOwnProperty("transferRecord")) {
                    if (!Array.isArray(message.transferRecord))
                        return "transferRecord: array expected";
                    for (var i = 0; i < message.transferRecord.length; ++i) {
                        var error = $root.serialization.TransferRecord.verify(message.transferRecord[i]);
                        if (error)
                            return "transferRecord." + error;
                    }
                }
                return null;
            };
    
            /**
             * Creates a ReservedFund message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.ReservedFund
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.ReservedFund} ReservedFund
             */
            ReservedFund.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.ReservedFund)
                    return object;
                var message = new $root.serialization.ReservedFund();
                if (object.collateral) {
                    if (!Array.isArray(object.collateral))
                        throw TypeError(".serialization.ReservedFund.collateral: array expected");
                    message.collateral = [];
                    for (var i = 0; i < object.collateral.length; ++i) {
                        if (typeof object.collateral[i] !== "object")
                            throw TypeError(".serialization.ReservedFund.collateral: object expected");
                        message.collateral[i] = $root.serialization.Coin.fromObject(object.collateral[i]);
                    }
                }
                if (object.initialFund) {
                    if (!Array.isArray(object.initialFund))
                        throw TypeError(".serialization.ReservedFund.initialFund: array expected");
                    message.initialFund = [];
                    for (var i = 0; i < object.initialFund.length; ++i) {
                        if (typeof object.initialFund[i] !== "object")
                            throw TypeError(".serialization.ReservedFund.initialFund: object expected");
                        message.initialFund[i] = $root.serialization.Coin.fromObject(object.initialFund[i]);
                    }
                }
                if (object.usedFund) {
                    if (!Array.isArray(object.usedFund))
                        throw TypeError(".serialization.ReservedFund.usedFund: array expected");
                    message.usedFund = [];
                    for (var i = 0; i < object.usedFund.length; ++i) {
                        if (typeof object.usedFund[i] !== "object")
                            throw TypeError(".serialization.ReservedFund.usedFund: object expected");
                        message.usedFund[i] = $root.serialization.Coin.fromObject(object.usedFund[i]);
                    }
                }
                if (object.resourceIDs) {
                    if (!Array.isArray(object.resourceIDs))
                        throw TypeError(".serialization.ReservedFund.resourceIDs: array expected");
                    message.resourceIDs = [];
                    for (var i = 0; i < object.resourceIDs.length; ++i)
                        if (typeof object.resourceIDs[i] === "string")
                            $util.base64.decode(object.resourceIDs[i], message.resourceIDs[i] = $util.newBuffer($util.base64.length(object.resourceIDs[i])), 0);
                        else if (object.resourceIDs[i].length)
                            message.resourceIDs[i] = object.resourceIDs[i];
                }
                if (object.endBlockHeight != null)
                    message.endBlockHeight = object.endBlockHeight | 0;
                if (object.reserveSequence != null)
                    message.reserveSequence = object.reserveSequence | 0;
                if (object.transferRecord) {
                    if (!Array.isArray(object.transferRecord))
                        throw TypeError(".serialization.ReservedFund.transferRecord: array expected");
                    message.transferRecord = [];
                    for (var i = 0; i < object.transferRecord.length; ++i) {
                        if (typeof object.transferRecord[i] !== "object")
                            throw TypeError(".serialization.ReservedFund.transferRecord: object expected");
                        message.transferRecord[i] = $root.serialization.TransferRecord.fromObject(object.transferRecord[i]);
                    }
                }
                return message;
            };
    
            /**
             * Creates a plain object from a ReservedFund message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.ReservedFund
             * @static
             * @param {serialization.ReservedFund} message ReservedFund
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ReservedFund.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.arrays || options.defaults) {
                    object.collateral = [];
                    object.initialFund = [];
                    object.usedFund = [];
                    object.resourceIDs = [];
                    object.transferRecord = [];
                }
                if (options.defaults) {
                    object.endBlockHeight = 0;
                    object.reserveSequence = 0;
                }
                if (message.collateral && message.collateral.length) {
                    object.collateral = [];
                    for (var j = 0; j < message.collateral.length; ++j)
                        object.collateral[j] = $root.serialization.Coin.toObject(message.collateral[j], options);
                }
                if (message.initialFund && message.initialFund.length) {
                    object.initialFund = [];
                    for (var j = 0; j < message.initialFund.length; ++j)
                        object.initialFund[j] = $root.serialization.Coin.toObject(message.initialFund[j], options);
                }
                if (message.usedFund && message.usedFund.length) {
                    object.usedFund = [];
                    for (var j = 0; j < message.usedFund.length; ++j)
                        object.usedFund[j] = $root.serialization.Coin.toObject(message.usedFund[j], options);
                }
                if (message.resourceIDs && message.resourceIDs.length) {
                    object.resourceIDs = [];
                    for (var j = 0; j < message.resourceIDs.length; ++j)
                        object.resourceIDs[j] = options.bytes === String ? $util.base64.encode(message.resourceIDs[j], 0, message.resourceIDs[j].length) : options.bytes === Array ? Array.prototype.slice.call(message.resourceIDs[j]) : message.resourceIDs[j];
                }
                if (message.endBlockHeight != null && message.hasOwnProperty("endBlockHeight"))
                    object.endBlockHeight = message.endBlockHeight;
                if (message.reserveSequence != null && message.hasOwnProperty("reserveSequence"))
                    object.reserveSequence = message.reserveSequence;
                if (message.transferRecord && message.transferRecord.length) {
                    object.transferRecord = [];
                    for (var j = 0; j < message.transferRecord.length; ++j)
                        object.transferRecord[j] = $root.serialization.TransferRecord.toObject(message.transferRecord[j], options);
                }
                return object;
            };
    
            /**
             * Converts this ReservedFund to JSON.
             * @function toJSON
             * @memberof serialization.ReservedFund
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ReservedFund.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return ReservedFund;
        })();
    
        serialization.TransferRecord = (function() {
    
            /**
             * Properties of a TransferRecord.
             * @memberof serialization
             * @interface ITransferRecord
             * @property {serialization.IServicePaymentTx|null} [servicePayment] TransferRecord servicePayment
             */
    
            /**
             * Constructs a new TransferRecord.
             * @memberof serialization
             * @classdesc Represents a TransferRecord.
             * @implements ITransferRecord
             * @constructor
             * @param {serialization.ITransferRecord=} [properties] Properties to set
             */
            function TransferRecord(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * TransferRecord servicePayment.
             * @member {serialization.IServicePaymentTx|null|undefined} servicePayment
             * @memberof serialization.TransferRecord
             * @instance
             */
            TransferRecord.prototype.servicePayment = null;
    
            /**
             * Creates a new TransferRecord instance using the specified properties.
             * @function create
             * @memberof serialization.TransferRecord
             * @static
             * @param {serialization.ITransferRecord=} [properties] Properties to set
             * @returns {serialization.TransferRecord} TransferRecord instance
             */
            TransferRecord.create = function create(properties) {
                return new TransferRecord(properties);
            };
    
            /**
             * Encodes the specified TransferRecord message. Does not implicitly {@link serialization.TransferRecord.verify|verify} messages.
             * @function encode
             * @memberof serialization.TransferRecord
             * @static
             * @param {serialization.ITransferRecord} message TransferRecord message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            TransferRecord.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.servicePayment != null && message.hasOwnProperty("servicePayment"))
                    $root.serialization.ServicePaymentTx.encode(message.servicePayment, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                return writer;
            };
    
            /**
             * Encodes the specified TransferRecord message, length delimited. Does not implicitly {@link serialization.TransferRecord.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.TransferRecord
             * @static
             * @param {serialization.ITransferRecord} message TransferRecord message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            TransferRecord.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes a TransferRecord message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.TransferRecord
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.TransferRecord} TransferRecord
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            TransferRecord.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.TransferRecord();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.servicePayment = $root.serialization.ServicePaymentTx.decode(reader, reader.uint32());
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes a TransferRecord message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.TransferRecord
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.TransferRecord} TransferRecord
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            TransferRecord.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies a TransferRecord message.
             * @function verify
             * @memberof serialization.TransferRecord
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            TransferRecord.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.servicePayment != null && message.hasOwnProperty("servicePayment")) {
                    var error = $root.serialization.ServicePaymentTx.verify(message.servicePayment);
                    if (error)
                        return "servicePayment." + error;
                }
                return null;
            };
    
            /**
             * Creates a TransferRecord message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.TransferRecord
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.TransferRecord} TransferRecord
             */
            TransferRecord.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.TransferRecord)
                    return object;
                var message = new $root.serialization.TransferRecord();
                if (object.servicePayment != null) {
                    if (typeof object.servicePayment !== "object")
                        throw TypeError(".serialization.TransferRecord.servicePayment: object expected");
                    message.servicePayment = $root.serialization.ServicePaymentTx.fromObject(object.servicePayment);
                }
                return message;
            };
    
            /**
             * Creates a plain object from a TransferRecord message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.TransferRecord
             * @static
             * @param {serialization.TransferRecord} message TransferRecord
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            TransferRecord.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults)
                    object.servicePayment = null;
                if (message.servicePayment != null && message.hasOwnProperty("servicePayment"))
                    object.servicePayment = $root.serialization.ServicePaymentTx.toObject(message.servicePayment, options);
                return object;
            };
    
            /**
             * Converts this TransferRecord to JSON.
             * @function toJSON
             * @memberof serialization.TransferRecord
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            TransferRecord.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return TransferRecord;
        })();
    
        serialization.Account = (function() {
    
            /**
             * Properties of an Account.
             * @memberof serialization
             * @interface IAccount
             * @property {number|Long|null} [sequence] Account sequence
             * @property {Array.<serialization.ICoin>|null} [balance] Account balance
             * @property {Array.<serialization.IReservedFund>|null} [reservedFunds] Account reservedFunds
             * @property {number|null} [lastUpdatedBlockHeight] Account lastUpdatedBlockHeight
             * @property {serialization.IPublicKey|null} [pubKey] Account pubKey
             */
    
            /**
             * Constructs a new Account.
             * @memberof serialization
             * @classdesc Represents an Account.
             * @implements IAccount
             * @constructor
             * @param {serialization.IAccount=} [properties] Properties to set
             */
            function Account(properties) {
                this.balance = [];
                this.reservedFunds = [];
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * Account sequence.
             * @member {number|Long} sequence
             * @memberof serialization.Account
             * @instance
             */
            Account.prototype.sequence = $util.Long ? $util.Long.fromBits(0,0,false) : 0;
    
            /**
             * Account balance.
             * @member {Array.<serialization.ICoin>} balance
             * @memberof serialization.Account
             * @instance
             */
            Account.prototype.balance = $util.emptyArray;
    
            /**
             * Account reservedFunds.
             * @member {Array.<serialization.IReservedFund>} reservedFunds
             * @memberof serialization.Account
             * @instance
             */
            Account.prototype.reservedFunds = $util.emptyArray;
    
            /**
             * Account lastUpdatedBlockHeight.
             * @member {number} lastUpdatedBlockHeight
             * @memberof serialization.Account
             * @instance
             */
            Account.prototype.lastUpdatedBlockHeight = 0;
    
            /**
             * Account pubKey.
             * @member {serialization.IPublicKey|null|undefined} pubKey
             * @memberof serialization.Account
             * @instance
             */
            Account.prototype.pubKey = null;
    
            /**
             * Creates a new Account instance using the specified properties.
             * @function create
             * @memberof serialization.Account
             * @static
             * @param {serialization.IAccount=} [properties] Properties to set
             * @returns {serialization.Account} Account instance
             */
            Account.create = function create(properties) {
                return new Account(properties);
            };
    
            /**
             * Encodes the specified Account message. Does not implicitly {@link serialization.Account.verify|verify} messages.
             * @function encode
             * @memberof serialization.Account
             * @static
             * @param {serialization.IAccount} message Account message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Account.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.sequence != null && message.hasOwnProperty("sequence"))
                    writer.uint32(/* id 1, wireType 0 =*/8).int64(message.sequence);
                if (message.balance != null && message.balance.length)
                    for (var i = 0; i < message.balance.length; ++i)
                        $root.serialization.Coin.encode(message.balance[i], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
                if (message.reservedFunds != null && message.reservedFunds.length)
                    for (var i = 0; i < message.reservedFunds.length; ++i)
                        $root.serialization.ReservedFund.encode(message.reservedFunds[i], writer.uint32(/* id 3, wireType 2 =*/26).fork()).ldelim();
                if (message.lastUpdatedBlockHeight != null && message.hasOwnProperty("lastUpdatedBlockHeight"))
                    writer.uint32(/* id 4, wireType 0 =*/32).int32(message.lastUpdatedBlockHeight);
                if (message.pubKey != null && message.hasOwnProperty("pubKey"))
                    $root.serialization.PublicKey.encode(message.pubKey, writer.uint32(/* id 5, wireType 2 =*/42).fork()).ldelim();
                return writer;
            };
    
            /**
             * Encodes the specified Account message, length delimited. Does not implicitly {@link serialization.Account.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.Account
             * @static
             * @param {serialization.IAccount} message Account message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Account.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes an Account message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.Account
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.Account} Account
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Account.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.Account();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.sequence = reader.int64();
                        break;
                    case 2:
                        if (!(message.balance && message.balance.length))
                            message.balance = [];
                        message.balance.push($root.serialization.Coin.decode(reader, reader.uint32()));
                        break;
                    case 3:
                        if (!(message.reservedFunds && message.reservedFunds.length))
                            message.reservedFunds = [];
                        message.reservedFunds.push($root.serialization.ReservedFund.decode(reader, reader.uint32()));
                        break;
                    case 4:
                        message.lastUpdatedBlockHeight = reader.int32();
                        break;
                    case 5:
                        message.pubKey = $root.serialization.PublicKey.decode(reader, reader.uint32());
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes an Account message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.Account
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.Account} Account
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Account.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies an Account message.
             * @function verify
             * @memberof serialization.Account
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            Account.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.sequence != null && message.hasOwnProperty("sequence"))
                    if (!$util.isInteger(message.sequence) && !(message.sequence && $util.isInteger(message.sequence.low) && $util.isInteger(message.sequence.high)))
                        return "sequence: integer|Long expected";
                if (message.balance != null && message.hasOwnProperty("balance")) {
                    if (!Array.isArray(message.balance))
                        return "balance: array expected";
                    for (var i = 0; i < message.balance.length; ++i) {
                        var error = $root.serialization.Coin.verify(message.balance[i]);
                        if (error)
                            return "balance." + error;
                    }
                }
                if (message.reservedFunds != null && message.hasOwnProperty("reservedFunds")) {
                    if (!Array.isArray(message.reservedFunds))
                        return "reservedFunds: array expected";
                    for (var i = 0; i < message.reservedFunds.length; ++i) {
                        var error = $root.serialization.ReservedFund.verify(message.reservedFunds[i]);
                        if (error)
                            return "reservedFunds." + error;
                    }
                }
                if (message.lastUpdatedBlockHeight != null && message.hasOwnProperty("lastUpdatedBlockHeight"))
                    if (!$util.isInteger(message.lastUpdatedBlockHeight))
                        return "lastUpdatedBlockHeight: integer expected";
                if (message.pubKey != null && message.hasOwnProperty("pubKey")) {
                    var error = $root.serialization.PublicKey.verify(message.pubKey);
                    if (error)
                        return "pubKey." + error;
                }
                return null;
            };
    
            /**
             * Creates an Account message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.Account
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.Account} Account
             */
            Account.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.Account)
                    return object;
                var message = new $root.serialization.Account();
                if (object.sequence != null)
                    if ($util.Long)
                        (message.sequence = $util.Long.fromValue(object.sequence)).unsigned = false;
                    else if (typeof object.sequence === "string")
                        message.sequence = parseInt(object.sequence, 10);
                    else if (typeof object.sequence === "number")
                        message.sequence = object.sequence;
                    else if (typeof object.sequence === "object")
                        message.sequence = new $util.LongBits(object.sequence.low >>> 0, object.sequence.high >>> 0).toNumber();
                if (object.balance) {
                    if (!Array.isArray(object.balance))
                        throw TypeError(".serialization.Account.balance: array expected");
                    message.balance = [];
                    for (var i = 0; i < object.balance.length; ++i) {
                        if (typeof object.balance[i] !== "object")
                            throw TypeError(".serialization.Account.balance: object expected");
                        message.balance[i] = $root.serialization.Coin.fromObject(object.balance[i]);
                    }
                }
                if (object.reservedFunds) {
                    if (!Array.isArray(object.reservedFunds))
                        throw TypeError(".serialization.Account.reservedFunds: array expected");
                    message.reservedFunds = [];
                    for (var i = 0; i < object.reservedFunds.length; ++i) {
                        if (typeof object.reservedFunds[i] !== "object")
                            throw TypeError(".serialization.Account.reservedFunds: object expected");
                        message.reservedFunds[i] = $root.serialization.ReservedFund.fromObject(object.reservedFunds[i]);
                    }
                }
                if (object.lastUpdatedBlockHeight != null)
                    message.lastUpdatedBlockHeight = object.lastUpdatedBlockHeight | 0;
                if (object.pubKey != null) {
                    if (typeof object.pubKey !== "object")
                        throw TypeError(".serialization.Account.pubKey: object expected");
                    message.pubKey = $root.serialization.PublicKey.fromObject(object.pubKey);
                }
                return message;
            };
    
            /**
             * Creates a plain object from an Account message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.Account
             * @static
             * @param {serialization.Account} message Account
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Account.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.arrays || options.defaults) {
                    object.balance = [];
                    object.reservedFunds = [];
                }
                if (options.defaults) {
                    if ($util.Long) {
                        var long = new $util.Long(0, 0, false);
                        object.sequence = options.longs === String ? long.toString() : options.longs === Number ? long.toNumber() : long;
                    } else
                        object.sequence = options.longs === String ? "0" : 0;
                    object.lastUpdatedBlockHeight = 0;
                    object.pubKey = null;
                }
                if (message.sequence != null && message.hasOwnProperty("sequence"))
                    if (typeof message.sequence === "number")
                        object.sequence = options.longs === String ? String(message.sequence) : message.sequence;
                    else
                        object.sequence = options.longs === String ? $util.Long.prototype.toString.call(message.sequence) : options.longs === Number ? new $util.LongBits(message.sequence.low >>> 0, message.sequence.high >>> 0).toNumber() : message.sequence;
                if (message.balance && message.balance.length) {
                    object.balance = [];
                    for (var j = 0; j < message.balance.length; ++j)
                        object.balance[j] = $root.serialization.Coin.toObject(message.balance[j], options);
                }
                if (message.reservedFunds && message.reservedFunds.length) {
                    object.reservedFunds = [];
                    for (var j = 0; j < message.reservedFunds.length; ++j)
                        object.reservedFunds[j] = $root.serialization.ReservedFund.toObject(message.reservedFunds[j], options);
                }
                if (message.lastUpdatedBlockHeight != null && message.hasOwnProperty("lastUpdatedBlockHeight"))
                    object.lastUpdatedBlockHeight = message.lastUpdatedBlockHeight;
                if (message.pubKey != null && message.hasOwnProperty("pubKey"))
                    object.pubKey = $root.serialization.PublicKey.toObject(message.pubKey, options);
                return object;
            };
    
            /**
             * Converts this Account to JSON.
             * @function toJSON
             * @memberof serialization.Account
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Account.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return Account;
        })();
    
        serialization.TxInput = (function() {
    
            /**
             * Properties of a TxInput.
             * @memberof serialization
             * @interface ITxInput
             * @property {Uint8Array|null} [address] TxInput address
             * @property {Array.<serialization.ICoin>|null} [coins] TxInput coins
             * @property {number|Long|null} [sequence] TxInput sequence
             * @property {serialization.ISignature|null} [signature] TxInput signature
             * @property {serialization.IPublicKey|null} [pubkey] TxInput pubkey
             */
    
            /**
             * Constructs a new TxInput.
             * @memberof serialization
             * @classdesc Represents a TxInput.
             * @implements ITxInput
             * @constructor
             * @param {serialization.ITxInput=} [properties] Properties to set
             */
            function TxInput(properties) {
                this.coins = [];
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * TxInput address.
             * @member {Uint8Array} address
             * @memberof serialization.TxInput
             * @instance
             */
            TxInput.prototype.address = $util.newBuffer([]);
    
            /**
             * TxInput coins.
             * @member {Array.<serialization.ICoin>} coins
             * @memberof serialization.TxInput
             * @instance
             */
            TxInput.prototype.coins = $util.emptyArray;
    
            /**
             * TxInput sequence.
             * @member {number|Long} sequence
             * @memberof serialization.TxInput
             * @instance
             */
            TxInput.prototype.sequence = $util.Long ? $util.Long.fromBits(0,0,false) : 0;
    
            /**
             * TxInput signature.
             * @member {serialization.ISignature|null|undefined} signature
             * @memberof serialization.TxInput
             * @instance
             */
            TxInput.prototype.signature = null;
    
            /**
             * TxInput pubkey.
             * @member {serialization.IPublicKey|null|undefined} pubkey
             * @memberof serialization.TxInput
             * @instance
             */
            TxInput.prototype.pubkey = null;
    
            /**
             * Creates a new TxInput instance using the specified properties.
             * @function create
             * @memberof serialization.TxInput
             * @static
             * @param {serialization.ITxInput=} [properties] Properties to set
             * @returns {serialization.TxInput} TxInput instance
             */
            TxInput.create = function create(properties) {
                return new TxInput(properties);
            };
    
            /**
             * Encodes the specified TxInput message. Does not implicitly {@link serialization.TxInput.verify|verify} messages.
             * @function encode
             * @memberof serialization.TxInput
             * @static
             * @param {serialization.ITxInput} message TxInput message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            TxInput.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.address != null && message.hasOwnProperty("address"))
                    writer.uint32(/* id 1, wireType 2 =*/10).bytes(message.address);
                if (message.coins != null && message.coins.length)
                    for (var i = 0; i < message.coins.length; ++i)
                        $root.serialization.Coin.encode(message.coins[i], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
                if (message.sequence != null && message.hasOwnProperty("sequence"))
                    writer.uint32(/* id 3, wireType 0 =*/24).int64(message.sequence);
                if (message.signature != null && message.hasOwnProperty("signature"))
                    $root.serialization.Signature.encode(message.signature, writer.uint32(/* id 4, wireType 2 =*/34).fork()).ldelim();
                if (message.pubkey != null && message.hasOwnProperty("pubkey"))
                    $root.serialization.PublicKey.encode(message.pubkey, writer.uint32(/* id 5, wireType 2 =*/42).fork()).ldelim();
                return writer;
            };
    
            /**
             * Encodes the specified TxInput message, length delimited. Does not implicitly {@link serialization.TxInput.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.TxInput
             * @static
             * @param {serialization.ITxInput} message TxInput message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            TxInput.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes a TxInput message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.TxInput
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.TxInput} TxInput
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            TxInput.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.TxInput();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.address = reader.bytes();
                        break;
                    case 2:
                        if (!(message.coins && message.coins.length))
                            message.coins = [];
                        message.coins.push($root.serialization.Coin.decode(reader, reader.uint32()));
                        break;
                    case 3:
                        message.sequence = reader.int64();
                        break;
                    case 4:
                        message.signature = $root.serialization.Signature.decode(reader, reader.uint32());
                        break;
                    case 5:
                        message.pubkey = $root.serialization.PublicKey.decode(reader, reader.uint32());
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes a TxInput message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.TxInput
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.TxInput} TxInput
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            TxInput.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies a TxInput message.
             * @function verify
             * @memberof serialization.TxInput
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            TxInput.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.address != null && message.hasOwnProperty("address"))
                    if (!(message.address && typeof message.address.length === "number" || $util.isString(message.address)))
                        return "address: buffer expected";
                if (message.coins != null && message.hasOwnProperty("coins")) {
                    if (!Array.isArray(message.coins))
                        return "coins: array expected";
                    for (var i = 0; i < message.coins.length; ++i) {
                        var error = $root.serialization.Coin.verify(message.coins[i]);
                        if (error)
                            return "coins." + error;
                    }
                }
                if (message.sequence != null && message.hasOwnProperty("sequence"))
                    if (!$util.isInteger(message.sequence) && !(message.sequence && $util.isInteger(message.sequence.low) && $util.isInteger(message.sequence.high)))
                        return "sequence: integer|Long expected";
                if (message.signature != null && message.hasOwnProperty("signature")) {
                    var error = $root.serialization.Signature.verify(message.signature);
                    if (error)
                        return "signature." + error;
                }
                if (message.pubkey != null && message.hasOwnProperty("pubkey")) {
                    var error = $root.serialization.PublicKey.verify(message.pubkey);
                    if (error)
                        return "pubkey." + error;
                }
                return null;
            };
    
            /**
             * Creates a TxInput message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.TxInput
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.TxInput} TxInput
             */
            TxInput.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.TxInput)
                    return object;
                var message = new $root.serialization.TxInput();
                if (object.address != null)
                    if (typeof object.address === "string")
                        $util.base64.decode(object.address, message.address = $util.newBuffer($util.base64.length(object.address)), 0);
                    else if (object.address.length)
                        message.address = object.address;
                if (object.coins) {
                    if (!Array.isArray(object.coins))
                        throw TypeError(".serialization.TxInput.coins: array expected");
                    message.coins = [];
                    for (var i = 0; i < object.coins.length; ++i) {
                        if (typeof object.coins[i] !== "object")
                            throw TypeError(".serialization.TxInput.coins: object expected");
                        message.coins[i] = $root.serialization.Coin.fromObject(object.coins[i]);
                    }
                }
                if (object.sequence != null)
                    if ($util.Long)
                        (message.sequence = $util.Long.fromValue(object.sequence)).unsigned = false;
                    else if (typeof object.sequence === "string")
                        message.sequence = parseInt(object.sequence, 10);
                    else if (typeof object.sequence === "number")
                        message.sequence = object.sequence;
                    else if (typeof object.sequence === "object")
                        message.sequence = new $util.LongBits(object.sequence.low >>> 0, object.sequence.high >>> 0).toNumber();
                if (object.signature != null) {
                    if (typeof object.signature !== "object")
                        throw TypeError(".serialization.TxInput.signature: object expected");
                    message.signature = $root.serialization.Signature.fromObject(object.signature);
                }
                if (object.pubkey != null) {
                    if (typeof object.pubkey !== "object")
                        throw TypeError(".serialization.TxInput.pubkey: object expected");
                    message.pubkey = $root.serialization.PublicKey.fromObject(object.pubkey);
                }
                return message;
            };
    
            /**
             * Creates a plain object from a TxInput message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.TxInput
             * @static
             * @param {serialization.TxInput} message TxInput
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            TxInput.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.arrays || options.defaults)
                    object.coins = [];
                if (options.defaults) {
                    object.address = options.bytes === String ? "" : [];
                    if ($util.Long) {
                        var long = new $util.Long(0, 0, false);
                        object.sequence = options.longs === String ? long.toString() : options.longs === Number ? long.toNumber() : long;
                    } else
                        object.sequence = options.longs === String ? "0" : 0;
                    object.signature = null;
                    object.pubkey = null;
                }
                if (message.address != null && message.hasOwnProperty("address"))
                    object.address = options.bytes === String ? $util.base64.encode(message.address, 0, message.address.length) : options.bytes === Array ? Array.prototype.slice.call(message.address) : message.address;
                if (message.coins && message.coins.length) {
                    object.coins = [];
                    for (var j = 0; j < message.coins.length; ++j)
                        object.coins[j] = $root.serialization.Coin.toObject(message.coins[j], options);
                }
                if (message.sequence != null && message.hasOwnProperty("sequence"))
                    if (typeof message.sequence === "number")
                        object.sequence = options.longs === String ? String(message.sequence) : message.sequence;
                    else
                        object.sequence = options.longs === String ? $util.Long.prototype.toString.call(message.sequence) : options.longs === Number ? new $util.LongBits(message.sequence.low >>> 0, message.sequence.high >>> 0).toNumber() : message.sequence;
                if (message.signature != null && message.hasOwnProperty("signature"))
                    object.signature = $root.serialization.Signature.toObject(message.signature, options);
                if (message.pubkey != null && message.hasOwnProperty("pubkey"))
                    object.pubkey = $root.serialization.PublicKey.toObject(message.pubkey, options);
                return object;
            };
    
            /**
             * Converts this TxInput to JSON.
             * @function toJSON
             * @memberof serialization.TxInput
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            TxInput.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return TxInput;
        })();
    
        serialization.TxOutput = (function() {
    
            /**
             * Properties of a TxOutput.
             * @memberof serialization
             * @interface ITxOutput
             * @property {Uint8Array|null} [address] TxOutput address
             * @property {Array.<serialization.ICoin>|null} [coins] TxOutput coins
             */
    
            /**
             * Constructs a new TxOutput.
             * @memberof serialization
             * @classdesc Represents a TxOutput.
             * @implements ITxOutput
             * @constructor
             * @param {serialization.ITxOutput=} [properties] Properties to set
             */
            function TxOutput(properties) {
                this.coins = [];
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * TxOutput address.
             * @member {Uint8Array} address
             * @memberof serialization.TxOutput
             * @instance
             */
            TxOutput.prototype.address = $util.newBuffer([]);
    
            /**
             * TxOutput coins.
             * @member {Array.<serialization.ICoin>} coins
             * @memberof serialization.TxOutput
             * @instance
             */
            TxOutput.prototype.coins = $util.emptyArray;
    
            /**
             * Creates a new TxOutput instance using the specified properties.
             * @function create
             * @memberof serialization.TxOutput
             * @static
             * @param {serialization.ITxOutput=} [properties] Properties to set
             * @returns {serialization.TxOutput} TxOutput instance
             */
            TxOutput.create = function create(properties) {
                return new TxOutput(properties);
            };
    
            /**
             * Encodes the specified TxOutput message. Does not implicitly {@link serialization.TxOutput.verify|verify} messages.
             * @function encode
             * @memberof serialization.TxOutput
             * @static
             * @param {serialization.ITxOutput} message TxOutput message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            TxOutput.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.address != null && message.hasOwnProperty("address"))
                    writer.uint32(/* id 1, wireType 2 =*/10).bytes(message.address);
                if (message.coins != null && message.coins.length)
                    for (var i = 0; i < message.coins.length; ++i)
                        $root.serialization.Coin.encode(message.coins[i], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
                return writer;
            };
    
            /**
             * Encodes the specified TxOutput message, length delimited. Does not implicitly {@link serialization.TxOutput.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.TxOutput
             * @static
             * @param {serialization.ITxOutput} message TxOutput message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            TxOutput.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes a TxOutput message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.TxOutput
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.TxOutput} TxOutput
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            TxOutput.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.TxOutput();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.address = reader.bytes();
                        break;
                    case 2:
                        if (!(message.coins && message.coins.length))
                            message.coins = [];
                        message.coins.push($root.serialization.Coin.decode(reader, reader.uint32()));
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes a TxOutput message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.TxOutput
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.TxOutput} TxOutput
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            TxOutput.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies a TxOutput message.
             * @function verify
             * @memberof serialization.TxOutput
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            TxOutput.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.address != null && message.hasOwnProperty("address"))
                    if (!(message.address && typeof message.address.length === "number" || $util.isString(message.address)))
                        return "address: buffer expected";
                if (message.coins != null && message.hasOwnProperty("coins")) {
                    if (!Array.isArray(message.coins))
                        return "coins: array expected";
                    for (var i = 0; i < message.coins.length; ++i) {
                        var error = $root.serialization.Coin.verify(message.coins[i]);
                        if (error)
                            return "coins." + error;
                    }
                }
                return null;
            };
    
            /**
             * Creates a TxOutput message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.TxOutput
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.TxOutput} TxOutput
             */
            TxOutput.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.TxOutput)
                    return object;
                var message = new $root.serialization.TxOutput();
                if (object.address != null)
                    if (typeof object.address === "string")
                        $util.base64.decode(object.address, message.address = $util.newBuffer($util.base64.length(object.address)), 0);
                    else if (object.address.length)
                        message.address = object.address;
                if (object.coins) {
                    if (!Array.isArray(object.coins))
                        throw TypeError(".serialization.TxOutput.coins: array expected");
                    message.coins = [];
                    for (var i = 0; i < object.coins.length; ++i) {
                        if (typeof object.coins[i] !== "object")
                            throw TypeError(".serialization.TxOutput.coins: object expected");
                        message.coins[i] = $root.serialization.Coin.fromObject(object.coins[i]);
                    }
                }
                return message;
            };
    
            /**
             * Creates a plain object from a TxOutput message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.TxOutput
             * @static
             * @param {serialization.TxOutput} message TxOutput
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            TxOutput.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.arrays || options.defaults)
                    object.coins = [];
                if (options.defaults)
                    object.address = options.bytes === String ? "" : [];
                if (message.address != null && message.hasOwnProperty("address"))
                    object.address = options.bytes === String ? $util.base64.encode(message.address, 0, message.address.length) : options.bytes === Array ? Array.prototype.slice.call(message.address) : message.address;
                if (message.coins && message.coins.length) {
                    object.coins = [];
                    for (var j = 0; j < message.coins.length; ++j)
                        object.coins[j] = $root.serialization.Coin.toObject(message.coins[j], options);
                }
                return object;
            };
    
            /**
             * Converts this TxOutput to JSON.
             * @function toJSON
             * @memberof serialization.TxOutput
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            TxOutput.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return TxOutput;
        })();
    
        serialization.Tx = (function() {
    
            /**
             * Properties of a Tx.
             * @memberof serialization
             * @interface ITx
             * @property {serialization.ICoinbaseTx|null} [coinbase] Tx coinbase
             * @property {serialization.ISendTx|null} [send] Tx send
             * @property {serialization.IReserveFundTx|null} [reserve] Tx reserve
             * @property {serialization.IReleaseFundTx|null} [release] Tx release
             * @property {serialization.IServicePaymentTx|null} [servicePayment] Tx servicePayment
             * @property {serialization.ISlashTx|null} [slash] Tx slash
             * @property {serialization.ISplitContractTx|null} [splitContract] Tx splitContract
             * @property {serialization.IUpdateValidatorsTx|null} [updateValidators] Tx updateValidators
             */
    
            /**
             * Constructs a new Tx.
             * @memberof serialization
             * @classdesc Represents a Tx.
             * @implements ITx
             * @constructor
             * @param {serialization.ITx=} [properties] Properties to set
             */
            function Tx(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * Tx coinbase.
             * @member {serialization.ICoinbaseTx|null|undefined} coinbase
             * @memberof serialization.Tx
             * @instance
             */
            Tx.prototype.coinbase = null;
    
            /**
             * Tx send.
             * @member {serialization.ISendTx|null|undefined} send
             * @memberof serialization.Tx
             * @instance
             */
            Tx.prototype.send = null;
    
            /**
             * Tx reserve.
             * @member {serialization.IReserveFundTx|null|undefined} reserve
             * @memberof serialization.Tx
             * @instance
             */
            Tx.prototype.reserve = null;
    
            /**
             * Tx release.
             * @member {serialization.IReleaseFundTx|null|undefined} release
             * @memberof serialization.Tx
             * @instance
             */
            Tx.prototype.release = null;
    
            /**
             * Tx servicePayment.
             * @member {serialization.IServicePaymentTx|null|undefined} servicePayment
             * @memberof serialization.Tx
             * @instance
             */
            Tx.prototype.servicePayment = null;
    
            /**
             * Tx slash.
             * @member {serialization.ISlashTx|null|undefined} slash
             * @memberof serialization.Tx
             * @instance
             */
            Tx.prototype.slash = null;
    
            /**
             * Tx splitContract.
             * @member {serialization.ISplitContractTx|null|undefined} splitContract
             * @memberof serialization.Tx
             * @instance
             */
            Tx.prototype.splitContract = null;
    
            /**
             * Tx updateValidators.
             * @member {serialization.IUpdateValidatorsTx|null|undefined} updateValidators
             * @memberof serialization.Tx
             * @instance
             */
            Tx.prototype.updateValidators = null;
    
            // OneOf field names bound to virtual getters and setters
            var $oneOfFields;
    
            /**
             * Tx tx.
             * @member {"coinbase"|"send"|"reserve"|"release"|"servicePayment"|"slash"|"splitContract"|"updateValidators"|undefined} tx
             * @memberof serialization.Tx
             * @instance
             */
            Object.defineProperty(Tx.prototype, "tx", {
                get: $util.oneOfGetter($oneOfFields = ["coinbase", "send", "reserve", "release", "servicePayment", "slash", "splitContract", "updateValidators"]),
                set: $util.oneOfSetter($oneOfFields)
            });
    
            /**
             * Creates a new Tx instance using the specified properties.
             * @function create
             * @memberof serialization.Tx
             * @static
             * @param {serialization.ITx=} [properties] Properties to set
             * @returns {serialization.Tx} Tx instance
             */
            Tx.create = function create(properties) {
                return new Tx(properties);
            };
    
            /**
             * Encodes the specified Tx message. Does not implicitly {@link serialization.Tx.verify|verify} messages.
             * @function encode
             * @memberof serialization.Tx
             * @static
             * @param {serialization.ITx} message Tx message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Tx.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.coinbase != null && message.hasOwnProperty("coinbase"))
                    $root.serialization.CoinbaseTx.encode(message.coinbase, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                if (message.send != null && message.hasOwnProperty("send"))
                    $root.serialization.SendTx.encode(message.send, writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
                if (message.reserve != null && message.hasOwnProperty("reserve"))
                    $root.serialization.ReserveFundTx.encode(message.reserve, writer.uint32(/* id 3, wireType 2 =*/26).fork()).ldelim();
                if (message.release != null && message.hasOwnProperty("release"))
                    $root.serialization.ReleaseFundTx.encode(message.release, writer.uint32(/* id 4, wireType 2 =*/34).fork()).ldelim();
                if (message.servicePayment != null && message.hasOwnProperty("servicePayment"))
                    $root.serialization.ServicePaymentTx.encode(message.servicePayment, writer.uint32(/* id 5, wireType 2 =*/42).fork()).ldelim();
                if (message.slash != null && message.hasOwnProperty("slash"))
                    $root.serialization.SlashTx.encode(message.slash, writer.uint32(/* id 6, wireType 2 =*/50).fork()).ldelim();
                if (message.splitContract != null && message.hasOwnProperty("splitContract"))
                    $root.serialization.SplitContractTx.encode(message.splitContract, writer.uint32(/* id 7, wireType 2 =*/58).fork()).ldelim();
                if (message.updateValidators != null && message.hasOwnProperty("updateValidators"))
                    $root.serialization.UpdateValidatorsTx.encode(message.updateValidators, writer.uint32(/* id 8, wireType 2 =*/66).fork()).ldelim();
                return writer;
            };
    
            /**
             * Encodes the specified Tx message, length delimited. Does not implicitly {@link serialization.Tx.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.Tx
             * @static
             * @param {serialization.ITx} message Tx message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Tx.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes a Tx message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.Tx
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.Tx} Tx
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Tx.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.Tx();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.coinbase = $root.serialization.CoinbaseTx.decode(reader, reader.uint32());
                        break;
                    case 2:
                        message.send = $root.serialization.SendTx.decode(reader, reader.uint32());
                        break;
                    case 3:
                        message.reserve = $root.serialization.ReserveFundTx.decode(reader, reader.uint32());
                        break;
                    case 4:
                        message.release = $root.serialization.ReleaseFundTx.decode(reader, reader.uint32());
                        break;
                    case 5:
                        message.servicePayment = $root.serialization.ServicePaymentTx.decode(reader, reader.uint32());
                        break;
                    case 6:
                        message.slash = $root.serialization.SlashTx.decode(reader, reader.uint32());
                        break;
                    case 7:
                        message.splitContract = $root.serialization.SplitContractTx.decode(reader, reader.uint32());
                        break;
                    case 8:
                        message.updateValidators = $root.serialization.UpdateValidatorsTx.decode(reader, reader.uint32());
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes a Tx message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.Tx
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.Tx} Tx
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Tx.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies a Tx message.
             * @function verify
             * @memberof serialization.Tx
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            Tx.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                var properties = {};
                if (message.coinbase != null && message.hasOwnProperty("coinbase")) {
                    properties.tx = 1;
                    {
                        var error = $root.serialization.CoinbaseTx.verify(message.coinbase);
                        if (error)
                            return "coinbase." + error;
                    }
                }
                if (message.send != null && message.hasOwnProperty("send")) {
                    if (properties.tx === 1)
                        return "tx: multiple values";
                    properties.tx = 1;
                    {
                        var error = $root.serialization.SendTx.verify(message.send);
                        if (error)
                            return "send." + error;
                    }
                }
                if (message.reserve != null && message.hasOwnProperty("reserve")) {
                    if (properties.tx === 1)
                        return "tx: multiple values";
                    properties.tx = 1;
                    {
                        var error = $root.serialization.ReserveFundTx.verify(message.reserve);
                        if (error)
                            return "reserve." + error;
                    }
                }
                if (message.release != null && message.hasOwnProperty("release")) {
                    if (properties.tx === 1)
                        return "tx: multiple values";
                    properties.tx = 1;
                    {
                        var error = $root.serialization.ReleaseFundTx.verify(message.release);
                        if (error)
                            return "release." + error;
                    }
                }
                if (message.servicePayment != null && message.hasOwnProperty("servicePayment")) {
                    if (properties.tx === 1)
                        return "tx: multiple values";
                    properties.tx = 1;
                    {
                        var error = $root.serialization.ServicePaymentTx.verify(message.servicePayment);
                        if (error)
                            return "servicePayment." + error;
                    }
                }
                if (message.slash != null && message.hasOwnProperty("slash")) {
                    if (properties.tx === 1)
                        return "tx: multiple values";
                    properties.tx = 1;
                    {
                        var error = $root.serialization.SlashTx.verify(message.slash);
                        if (error)
                            return "slash." + error;
                    }
                }
                if (message.splitContract != null && message.hasOwnProperty("splitContract")) {
                    if (properties.tx === 1)
                        return "tx: multiple values";
                    properties.tx = 1;
                    {
                        var error = $root.serialization.SplitContractTx.verify(message.splitContract);
                        if (error)
                            return "splitContract." + error;
                    }
                }
                if (message.updateValidators != null && message.hasOwnProperty("updateValidators")) {
                    if (properties.tx === 1)
                        return "tx: multiple values";
                    properties.tx = 1;
                    {
                        var error = $root.serialization.UpdateValidatorsTx.verify(message.updateValidators);
                        if (error)
                            return "updateValidators." + error;
                    }
                }
                return null;
            };
    
            /**
             * Creates a Tx message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.Tx
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.Tx} Tx
             */
            Tx.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.Tx)
                    return object;
                var message = new $root.serialization.Tx();
                if (object.coinbase != null) {
                    if (typeof object.coinbase !== "object")
                        throw TypeError(".serialization.Tx.coinbase: object expected");
                    message.coinbase = $root.serialization.CoinbaseTx.fromObject(object.coinbase);
                }
                if (object.send != null) {
                    if (typeof object.send !== "object")
                        throw TypeError(".serialization.Tx.send: object expected");
                    message.send = $root.serialization.SendTx.fromObject(object.send);
                }
                if (object.reserve != null) {
                    if (typeof object.reserve !== "object")
                        throw TypeError(".serialization.Tx.reserve: object expected");
                    message.reserve = $root.serialization.ReserveFundTx.fromObject(object.reserve);
                }
                if (object.release != null) {
                    if (typeof object.release !== "object")
                        throw TypeError(".serialization.Tx.release: object expected");
                    message.release = $root.serialization.ReleaseFundTx.fromObject(object.release);
                }
                if (object.servicePayment != null) {
                    if (typeof object.servicePayment !== "object")
                        throw TypeError(".serialization.Tx.servicePayment: object expected");
                    message.servicePayment = $root.serialization.ServicePaymentTx.fromObject(object.servicePayment);
                }
                if (object.slash != null) {
                    if (typeof object.slash !== "object")
                        throw TypeError(".serialization.Tx.slash: object expected");
                    message.slash = $root.serialization.SlashTx.fromObject(object.slash);
                }
                if (object.splitContract != null) {
                    if (typeof object.splitContract !== "object")
                        throw TypeError(".serialization.Tx.splitContract: object expected");
                    message.splitContract = $root.serialization.SplitContractTx.fromObject(object.splitContract);
                }
                if (object.updateValidators != null) {
                    if (typeof object.updateValidators !== "object")
                        throw TypeError(".serialization.Tx.updateValidators: object expected");
                    message.updateValidators = $root.serialization.UpdateValidatorsTx.fromObject(object.updateValidators);
                }
                return message;
            };
    
            /**
             * Creates a plain object from a Tx message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.Tx
             * @static
             * @param {serialization.Tx} message Tx
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Tx.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (message.coinbase != null && message.hasOwnProperty("coinbase")) {
                    object.coinbase = $root.serialization.CoinbaseTx.toObject(message.coinbase, options);
                    if (options.oneofs)
                        object.tx = "coinbase";
                }
                if (message.send != null && message.hasOwnProperty("send")) {
                    object.send = $root.serialization.SendTx.toObject(message.send, options);
                    if (options.oneofs)
                        object.tx = "send";
                }
                if (message.reserve != null && message.hasOwnProperty("reserve")) {
                    object.reserve = $root.serialization.ReserveFundTx.toObject(message.reserve, options);
                    if (options.oneofs)
                        object.tx = "reserve";
                }
                if (message.release != null && message.hasOwnProperty("release")) {
                    object.release = $root.serialization.ReleaseFundTx.toObject(message.release, options);
                    if (options.oneofs)
                        object.tx = "release";
                }
                if (message.servicePayment != null && message.hasOwnProperty("servicePayment")) {
                    object.servicePayment = $root.serialization.ServicePaymentTx.toObject(message.servicePayment, options);
                    if (options.oneofs)
                        object.tx = "servicePayment";
                }
                if (message.slash != null && message.hasOwnProperty("slash")) {
                    object.slash = $root.serialization.SlashTx.toObject(message.slash, options);
                    if (options.oneofs)
                        object.tx = "slash";
                }
                if (message.splitContract != null && message.hasOwnProperty("splitContract")) {
                    object.splitContract = $root.serialization.SplitContractTx.toObject(message.splitContract, options);
                    if (options.oneofs)
                        object.tx = "splitContract";
                }
                if (message.updateValidators != null && message.hasOwnProperty("updateValidators")) {
                    object.updateValidators = $root.serialization.UpdateValidatorsTx.toObject(message.updateValidators, options);
                    if (options.oneofs)
                        object.tx = "updateValidators";
                }
                return object;
            };
    
            /**
             * Converts this Tx to JSON.
             * @function toJSON
             * @memberof serialization.Tx
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Tx.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return Tx;
        })();
    
        serialization.CoinbaseTx = (function() {
    
            /**
             * Properties of a CoinbaseTx.
             * @memberof serialization
             * @interface ICoinbaseTx
             * @property {serialization.ITxInput|null} [proposer] CoinbaseTx proposer
             * @property {Array.<serialization.ITxOutput>|null} [outputs] CoinbaseTx outputs
             * @property {number|null} [blockHeight] CoinbaseTx blockHeight
             */
    
            /**
             * Constructs a new CoinbaseTx.
             * @memberof serialization
             * @classdesc Represents a CoinbaseTx.
             * @implements ICoinbaseTx
             * @constructor
             * @param {serialization.ICoinbaseTx=} [properties] Properties to set
             */
            function CoinbaseTx(properties) {
                this.outputs = [];
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * CoinbaseTx proposer.
             * @member {serialization.ITxInput|null|undefined} proposer
             * @memberof serialization.CoinbaseTx
             * @instance
             */
            CoinbaseTx.prototype.proposer = null;
    
            /**
             * CoinbaseTx outputs.
             * @member {Array.<serialization.ITxOutput>} outputs
             * @memberof serialization.CoinbaseTx
             * @instance
             */
            CoinbaseTx.prototype.outputs = $util.emptyArray;
    
            /**
             * CoinbaseTx blockHeight.
             * @member {number} blockHeight
             * @memberof serialization.CoinbaseTx
             * @instance
             */
            CoinbaseTx.prototype.blockHeight = 0;
    
            /**
             * Creates a new CoinbaseTx instance using the specified properties.
             * @function create
             * @memberof serialization.CoinbaseTx
             * @static
             * @param {serialization.ICoinbaseTx=} [properties] Properties to set
             * @returns {serialization.CoinbaseTx} CoinbaseTx instance
             */
            CoinbaseTx.create = function create(properties) {
                return new CoinbaseTx(properties);
            };
    
            /**
             * Encodes the specified CoinbaseTx message. Does not implicitly {@link serialization.CoinbaseTx.verify|verify} messages.
             * @function encode
             * @memberof serialization.CoinbaseTx
             * @static
             * @param {serialization.ICoinbaseTx} message CoinbaseTx message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            CoinbaseTx.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.proposer != null && message.hasOwnProperty("proposer"))
                    $root.serialization.TxInput.encode(message.proposer, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                if (message.outputs != null && message.outputs.length)
                    for (var i = 0; i < message.outputs.length; ++i)
                        $root.serialization.TxOutput.encode(message.outputs[i], writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
                if (message.blockHeight != null && message.hasOwnProperty("blockHeight"))
                    writer.uint32(/* id 3, wireType 0 =*/24).int32(message.blockHeight);
                return writer;
            };
    
            /**
             * Encodes the specified CoinbaseTx message, length delimited. Does not implicitly {@link serialization.CoinbaseTx.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.CoinbaseTx
             * @static
             * @param {serialization.ICoinbaseTx} message CoinbaseTx message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            CoinbaseTx.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes a CoinbaseTx message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.CoinbaseTx
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.CoinbaseTx} CoinbaseTx
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            CoinbaseTx.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.CoinbaseTx();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.proposer = $root.serialization.TxInput.decode(reader, reader.uint32());
                        break;
                    case 2:
                        if (!(message.outputs && message.outputs.length))
                            message.outputs = [];
                        message.outputs.push($root.serialization.TxOutput.decode(reader, reader.uint32()));
                        break;
                    case 3:
                        message.blockHeight = reader.int32();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes a CoinbaseTx message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.CoinbaseTx
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.CoinbaseTx} CoinbaseTx
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            CoinbaseTx.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies a CoinbaseTx message.
             * @function verify
             * @memberof serialization.CoinbaseTx
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            CoinbaseTx.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.proposer != null && message.hasOwnProperty("proposer")) {
                    var error = $root.serialization.TxInput.verify(message.proposer);
                    if (error)
                        return "proposer." + error;
                }
                if (message.outputs != null && message.hasOwnProperty("outputs")) {
                    if (!Array.isArray(message.outputs))
                        return "outputs: array expected";
                    for (var i = 0; i < message.outputs.length; ++i) {
                        var error = $root.serialization.TxOutput.verify(message.outputs[i]);
                        if (error)
                            return "outputs." + error;
                    }
                }
                if (message.blockHeight != null && message.hasOwnProperty("blockHeight"))
                    if (!$util.isInteger(message.blockHeight))
                        return "blockHeight: integer expected";
                return null;
            };
    
            /**
             * Creates a CoinbaseTx message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.CoinbaseTx
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.CoinbaseTx} CoinbaseTx
             */
            CoinbaseTx.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.CoinbaseTx)
                    return object;
                var message = new $root.serialization.CoinbaseTx();
                if (object.proposer != null) {
                    if (typeof object.proposer !== "object")
                        throw TypeError(".serialization.CoinbaseTx.proposer: object expected");
                    message.proposer = $root.serialization.TxInput.fromObject(object.proposer);
                }
                if (object.outputs) {
                    if (!Array.isArray(object.outputs))
                        throw TypeError(".serialization.CoinbaseTx.outputs: array expected");
                    message.outputs = [];
                    for (var i = 0; i < object.outputs.length; ++i) {
                        if (typeof object.outputs[i] !== "object")
                            throw TypeError(".serialization.CoinbaseTx.outputs: object expected");
                        message.outputs[i] = $root.serialization.TxOutput.fromObject(object.outputs[i]);
                    }
                }
                if (object.blockHeight != null)
                    message.blockHeight = object.blockHeight | 0;
                return message;
            };
    
            /**
             * Creates a plain object from a CoinbaseTx message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.CoinbaseTx
             * @static
             * @param {serialization.CoinbaseTx} message CoinbaseTx
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            CoinbaseTx.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.arrays || options.defaults)
                    object.outputs = [];
                if (options.defaults) {
                    object.proposer = null;
                    object.blockHeight = 0;
                }
                if (message.proposer != null && message.hasOwnProperty("proposer"))
                    object.proposer = $root.serialization.TxInput.toObject(message.proposer, options);
                if (message.outputs && message.outputs.length) {
                    object.outputs = [];
                    for (var j = 0; j < message.outputs.length; ++j)
                        object.outputs[j] = $root.serialization.TxOutput.toObject(message.outputs[j], options);
                }
                if (message.blockHeight != null && message.hasOwnProperty("blockHeight"))
                    object.blockHeight = message.blockHeight;
                return object;
            };
    
            /**
             * Converts this CoinbaseTx to JSON.
             * @function toJSON
             * @memberof serialization.CoinbaseTx
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            CoinbaseTx.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return CoinbaseTx;
        })();
    
        serialization.SendTx = (function() {
    
            /**
             * Properties of a SendTx.
             * @memberof serialization
             * @interface ISendTx
             * @property {number|Long|null} [gas] SendTx gas
             * @property {serialization.ICoin|null} [fee] SendTx fee
             * @property {Array.<serialization.ITxInput>|null} [inputs] SendTx inputs
             * @property {Array.<serialization.ITxOutput>|null} [outputs] SendTx outputs
             */
    
            /**
             * Constructs a new SendTx.
             * @memberof serialization
             * @classdesc Represents a SendTx.
             * @implements ISendTx
             * @constructor
             * @param {serialization.ISendTx=} [properties] Properties to set
             */
            function SendTx(properties) {
                this.inputs = [];
                this.outputs = [];
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * SendTx gas.
             * @member {number|Long} gas
             * @memberof serialization.SendTx
             * @instance
             */
            SendTx.prototype.gas = $util.Long ? $util.Long.fromBits(0,0,false) : 0;
    
            /**
             * SendTx fee.
             * @member {serialization.ICoin|null|undefined} fee
             * @memberof serialization.SendTx
             * @instance
             */
            SendTx.prototype.fee = null;
    
            /**
             * SendTx inputs.
             * @member {Array.<serialization.ITxInput>} inputs
             * @memberof serialization.SendTx
             * @instance
             */
            SendTx.prototype.inputs = $util.emptyArray;
    
            /**
             * SendTx outputs.
             * @member {Array.<serialization.ITxOutput>} outputs
             * @memberof serialization.SendTx
             * @instance
             */
            SendTx.prototype.outputs = $util.emptyArray;
    
            /**
             * Creates a new SendTx instance using the specified properties.
             * @function create
             * @memberof serialization.SendTx
             * @static
             * @param {serialization.ISendTx=} [properties] Properties to set
             * @returns {serialization.SendTx} SendTx instance
             */
            SendTx.create = function create(properties) {
                return new SendTx(properties);
            };
    
            /**
             * Encodes the specified SendTx message. Does not implicitly {@link serialization.SendTx.verify|verify} messages.
             * @function encode
             * @memberof serialization.SendTx
             * @static
             * @param {serialization.ISendTx} message SendTx message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            SendTx.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.gas != null && message.hasOwnProperty("gas"))
                    writer.uint32(/* id 1, wireType 0 =*/8).int64(message.gas);
                if (message.fee != null && message.hasOwnProperty("fee"))
                    $root.serialization.Coin.encode(message.fee, writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
                if (message.inputs != null && message.inputs.length)
                    for (var i = 0; i < message.inputs.length; ++i)
                        $root.serialization.TxInput.encode(message.inputs[i], writer.uint32(/* id 3, wireType 2 =*/26).fork()).ldelim();
                if (message.outputs != null && message.outputs.length)
                    for (var i = 0; i < message.outputs.length; ++i)
                        $root.serialization.TxOutput.encode(message.outputs[i], writer.uint32(/* id 4, wireType 2 =*/34).fork()).ldelim();
                return writer;
            };
    
            /**
             * Encodes the specified SendTx message, length delimited. Does not implicitly {@link serialization.SendTx.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.SendTx
             * @static
             * @param {serialization.ISendTx} message SendTx message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            SendTx.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes a SendTx message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.SendTx
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.SendTx} SendTx
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            SendTx.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.SendTx();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.gas = reader.int64();
                        break;
                    case 2:
                        message.fee = $root.serialization.Coin.decode(reader, reader.uint32());
                        break;
                    case 3:
                        if (!(message.inputs && message.inputs.length))
                            message.inputs = [];
                        message.inputs.push($root.serialization.TxInput.decode(reader, reader.uint32()));
                        break;
                    case 4:
                        if (!(message.outputs && message.outputs.length))
                            message.outputs = [];
                        message.outputs.push($root.serialization.TxOutput.decode(reader, reader.uint32()));
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes a SendTx message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.SendTx
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.SendTx} SendTx
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            SendTx.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies a SendTx message.
             * @function verify
             * @memberof serialization.SendTx
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            SendTx.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.gas != null && message.hasOwnProperty("gas"))
                    if (!$util.isInteger(message.gas) && !(message.gas && $util.isInteger(message.gas.low) && $util.isInteger(message.gas.high)))
                        return "gas: integer|Long expected";
                if (message.fee != null && message.hasOwnProperty("fee")) {
                    var error = $root.serialization.Coin.verify(message.fee);
                    if (error)
                        return "fee." + error;
                }
                if (message.inputs != null && message.hasOwnProperty("inputs")) {
                    if (!Array.isArray(message.inputs))
                        return "inputs: array expected";
                    for (var i = 0; i < message.inputs.length; ++i) {
                        var error = $root.serialization.TxInput.verify(message.inputs[i]);
                        if (error)
                            return "inputs." + error;
                    }
                }
                if (message.outputs != null && message.hasOwnProperty("outputs")) {
                    if (!Array.isArray(message.outputs))
                        return "outputs: array expected";
                    for (var i = 0; i < message.outputs.length; ++i) {
                        var error = $root.serialization.TxOutput.verify(message.outputs[i]);
                        if (error)
                            return "outputs." + error;
                    }
                }
                return null;
            };
    
            /**
             * Creates a SendTx message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.SendTx
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.SendTx} SendTx
             */
            SendTx.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.SendTx)
                    return object;
                var message = new $root.serialization.SendTx();
                if (object.gas != null)
                    if ($util.Long)
                        (message.gas = $util.Long.fromValue(object.gas)).unsigned = false;
                    else if (typeof object.gas === "string")
                        message.gas = parseInt(object.gas, 10);
                    else if (typeof object.gas === "number")
                        message.gas = object.gas;
                    else if (typeof object.gas === "object")
                        message.gas = new $util.LongBits(object.gas.low >>> 0, object.gas.high >>> 0).toNumber();
                if (object.fee != null) {
                    if (typeof object.fee !== "object")
                        throw TypeError(".serialization.SendTx.fee: object expected");
                    message.fee = $root.serialization.Coin.fromObject(object.fee);
                }
                if (object.inputs) {
                    if (!Array.isArray(object.inputs))
                        throw TypeError(".serialization.SendTx.inputs: array expected");
                    message.inputs = [];
                    for (var i = 0; i < object.inputs.length; ++i) {
                        if (typeof object.inputs[i] !== "object")
                            throw TypeError(".serialization.SendTx.inputs: object expected");
                        message.inputs[i] = $root.serialization.TxInput.fromObject(object.inputs[i]);
                    }
                }
                if (object.outputs) {
                    if (!Array.isArray(object.outputs))
                        throw TypeError(".serialization.SendTx.outputs: array expected");
                    message.outputs = [];
                    for (var i = 0; i < object.outputs.length; ++i) {
                        if (typeof object.outputs[i] !== "object")
                            throw TypeError(".serialization.SendTx.outputs: object expected");
                        message.outputs[i] = $root.serialization.TxOutput.fromObject(object.outputs[i]);
                    }
                }
                return message;
            };
    
            /**
             * Creates a plain object from a SendTx message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.SendTx
             * @static
             * @param {serialization.SendTx} message SendTx
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            SendTx.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.arrays || options.defaults) {
                    object.inputs = [];
                    object.outputs = [];
                }
                if (options.defaults) {
                    if ($util.Long) {
                        var long = new $util.Long(0, 0, false);
                        object.gas = options.longs === String ? long.toString() : options.longs === Number ? long.toNumber() : long;
                    } else
                        object.gas = options.longs === String ? "0" : 0;
                    object.fee = null;
                }
                if (message.gas != null && message.hasOwnProperty("gas"))
                    if (typeof message.gas === "number")
                        object.gas = options.longs === String ? String(message.gas) : message.gas;
                    else
                        object.gas = options.longs === String ? $util.Long.prototype.toString.call(message.gas) : options.longs === Number ? new $util.LongBits(message.gas.low >>> 0, message.gas.high >>> 0).toNumber() : message.gas;
                if (message.fee != null && message.hasOwnProperty("fee"))
                    object.fee = $root.serialization.Coin.toObject(message.fee, options);
                if (message.inputs && message.inputs.length) {
                    object.inputs = [];
                    for (var j = 0; j < message.inputs.length; ++j)
                        object.inputs[j] = $root.serialization.TxInput.toObject(message.inputs[j], options);
                }
                if (message.outputs && message.outputs.length) {
                    object.outputs = [];
                    for (var j = 0; j < message.outputs.length; ++j)
                        object.outputs[j] = $root.serialization.TxOutput.toObject(message.outputs[j], options);
                }
                return object;
            };
    
            /**
             * Converts this SendTx to JSON.
             * @function toJSON
             * @memberof serialization.SendTx
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            SendTx.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return SendTx;
        })();
    
        serialization.ReserveFundTx = (function() {
    
            /**
             * Properties of a ReserveFundTx.
             * @memberof serialization
             * @interface IReserveFundTx
             * @property {number|Long|null} [gas] ReserveFundTx gas
             * @property {serialization.ICoin|null} [fee] ReserveFundTx fee
             * @property {serialization.ITxInput|null} [source] ReserveFundTx source
             * @property {Array.<serialization.ICoin>|null} [collateral] ReserveFundTx collateral
             * @property {Array.<Uint8Array>|null} [resourceIDs] ReserveFundTx resourceIDs
             * @property {number|Long|null} [duration] ReserveFundTx duration
             */
    
            /**
             * Constructs a new ReserveFundTx.
             * @memberof serialization
             * @classdesc Represents a ReserveFundTx.
             * @implements IReserveFundTx
             * @constructor
             * @param {serialization.IReserveFundTx=} [properties] Properties to set
             */
            function ReserveFundTx(properties) {
                this.collateral = [];
                this.resourceIDs = [];
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * ReserveFundTx gas.
             * @member {number|Long} gas
             * @memberof serialization.ReserveFundTx
             * @instance
             */
            ReserveFundTx.prototype.gas = $util.Long ? $util.Long.fromBits(0,0,false) : 0;
    
            /**
             * ReserveFundTx fee.
             * @member {serialization.ICoin|null|undefined} fee
             * @memberof serialization.ReserveFundTx
             * @instance
             */
            ReserveFundTx.prototype.fee = null;
    
            /**
             * ReserveFundTx source.
             * @member {serialization.ITxInput|null|undefined} source
             * @memberof serialization.ReserveFundTx
             * @instance
             */
            ReserveFundTx.prototype.source = null;
    
            /**
             * ReserveFundTx collateral.
             * @member {Array.<serialization.ICoin>} collateral
             * @memberof serialization.ReserveFundTx
             * @instance
             */
            ReserveFundTx.prototype.collateral = $util.emptyArray;
    
            /**
             * ReserveFundTx resourceIDs.
             * @member {Array.<Uint8Array>} resourceIDs
             * @memberof serialization.ReserveFundTx
             * @instance
             */
            ReserveFundTx.prototype.resourceIDs = $util.emptyArray;
    
            /**
             * ReserveFundTx duration.
             * @member {number|Long} duration
             * @memberof serialization.ReserveFundTx
             * @instance
             */
            ReserveFundTx.prototype.duration = $util.Long ? $util.Long.fromBits(0,0,false) : 0;
    
            /**
             * Creates a new ReserveFundTx instance using the specified properties.
             * @function create
             * @memberof serialization.ReserveFundTx
             * @static
             * @param {serialization.IReserveFundTx=} [properties] Properties to set
             * @returns {serialization.ReserveFundTx} ReserveFundTx instance
             */
            ReserveFundTx.create = function create(properties) {
                return new ReserveFundTx(properties);
            };
    
            /**
             * Encodes the specified ReserveFundTx message. Does not implicitly {@link serialization.ReserveFundTx.verify|verify} messages.
             * @function encode
             * @memberof serialization.ReserveFundTx
             * @static
             * @param {serialization.IReserveFundTx} message ReserveFundTx message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ReserveFundTx.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.gas != null && message.hasOwnProperty("gas"))
                    writer.uint32(/* id 1, wireType 0 =*/8).int64(message.gas);
                if (message.fee != null && message.hasOwnProperty("fee"))
                    $root.serialization.Coin.encode(message.fee, writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
                if (message.source != null && message.hasOwnProperty("source"))
                    $root.serialization.TxInput.encode(message.source, writer.uint32(/* id 3, wireType 2 =*/26).fork()).ldelim();
                if (message.collateral != null && message.collateral.length)
                    for (var i = 0; i < message.collateral.length; ++i)
                        $root.serialization.Coin.encode(message.collateral[i], writer.uint32(/* id 4, wireType 2 =*/34).fork()).ldelim();
                if (message.resourceIDs != null && message.resourceIDs.length)
                    for (var i = 0; i < message.resourceIDs.length; ++i)
                        writer.uint32(/* id 5, wireType 2 =*/42).bytes(message.resourceIDs[i]);
                if (message.duration != null && message.hasOwnProperty("duration"))
                    writer.uint32(/* id 6, wireType 0 =*/48).int64(message.duration);
                return writer;
            };
    
            /**
             * Encodes the specified ReserveFundTx message, length delimited. Does not implicitly {@link serialization.ReserveFundTx.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.ReserveFundTx
             * @static
             * @param {serialization.IReserveFundTx} message ReserveFundTx message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ReserveFundTx.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes a ReserveFundTx message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.ReserveFundTx
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.ReserveFundTx} ReserveFundTx
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ReserveFundTx.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.ReserveFundTx();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.gas = reader.int64();
                        break;
                    case 2:
                        message.fee = $root.serialization.Coin.decode(reader, reader.uint32());
                        break;
                    case 3:
                        message.source = $root.serialization.TxInput.decode(reader, reader.uint32());
                        break;
                    case 4:
                        if (!(message.collateral && message.collateral.length))
                            message.collateral = [];
                        message.collateral.push($root.serialization.Coin.decode(reader, reader.uint32()));
                        break;
                    case 5:
                        if (!(message.resourceIDs && message.resourceIDs.length))
                            message.resourceIDs = [];
                        message.resourceIDs.push(reader.bytes());
                        break;
                    case 6:
                        message.duration = reader.int64();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes a ReserveFundTx message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.ReserveFundTx
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.ReserveFundTx} ReserveFundTx
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ReserveFundTx.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies a ReserveFundTx message.
             * @function verify
             * @memberof serialization.ReserveFundTx
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ReserveFundTx.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.gas != null && message.hasOwnProperty("gas"))
                    if (!$util.isInteger(message.gas) && !(message.gas && $util.isInteger(message.gas.low) && $util.isInteger(message.gas.high)))
                        return "gas: integer|Long expected";
                if (message.fee != null && message.hasOwnProperty("fee")) {
                    var error = $root.serialization.Coin.verify(message.fee);
                    if (error)
                        return "fee." + error;
                }
                if (message.source != null && message.hasOwnProperty("source")) {
                    var error = $root.serialization.TxInput.verify(message.source);
                    if (error)
                        return "source." + error;
                }
                if (message.collateral != null && message.hasOwnProperty("collateral")) {
                    if (!Array.isArray(message.collateral))
                        return "collateral: array expected";
                    for (var i = 0; i < message.collateral.length; ++i) {
                        var error = $root.serialization.Coin.verify(message.collateral[i]);
                        if (error)
                            return "collateral." + error;
                    }
                }
                if (message.resourceIDs != null && message.hasOwnProperty("resourceIDs")) {
                    if (!Array.isArray(message.resourceIDs))
                        return "resourceIDs: array expected";
                    for (var i = 0; i < message.resourceIDs.length; ++i)
                        if (!(message.resourceIDs[i] && typeof message.resourceIDs[i].length === "number" || $util.isString(message.resourceIDs[i])))
                            return "resourceIDs: buffer[] expected";
                }
                if (message.duration != null && message.hasOwnProperty("duration"))
                    if (!$util.isInteger(message.duration) && !(message.duration && $util.isInteger(message.duration.low) && $util.isInteger(message.duration.high)))
                        return "duration: integer|Long expected";
                return null;
            };
    
            /**
             * Creates a ReserveFundTx message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.ReserveFundTx
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.ReserveFundTx} ReserveFundTx
             */
            ReserveFundTx.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.ReserveFundTx)
                    return object;
                var message = new $root.serialization.ReserveFundTx();
                if (object.gas != null)
                    if ($util.Long)
                        (message.gas = $util.Long.fromValue(object.gas)).unsigned = false;
                    else if (typeof object.gas === "string")
                        message.gas = parseInt(object.gas, 10);
                    else if (typeof object.gas === "number")
                        message.gas = object.gas;
                    else if (typeof object.gas === "object")
                        message.gas = new $util.LongBits(object.gas.low >>> 0, object.gas.high >>> 0).toNumber();
                if (object.fee != null) {
                    if (typeof object.fee !== "object")
                        throw TypeError(".serialization.ReserveFundTx.fee: object expected");
                    message.fee = $root.serialization.Coin.fromObject(object.fee);
                }
                if (object.source != null) {
                    if (typeof object.source !== "object")
                        throw TypeError(".serialization.ReserveFundTx.source: object expected");
                    message.source = $root.serialization.TxInput.fromObject(object.source);
                }
                if (object.collateral) {
                    if (!Array.isArray(object.collateral))
                        throw TypeError(".serialization.ReserveFundTx.collateral: array expected");
                    message.collateral = [];
                    for (var i = 0; i < object.collateral.length; ++i) {
                        if (typeof object.collateral[i] !== "object")
                            throw TypeError(".serialization.ReserveFundTx.collateral: object expected");
                        message.collateral[i] = $root.serialization.Coin.fromObject(object.collateral[i]);
                    }
                }
                if (object.resourceIDs) {
                    if (!Array.isArray(object.resourceIDs))
                        throw TypeError(".serialization.ReserveFundTx.resourceIDs: array expected");
                    message.resourceIDs = [];
                    for (var i = 0; i < object.resourceIDs.length; ++i)
                        if (typeof object.resourceIDs[i] === "string")
                            $util.base64.decode(object.resourceIDs[i], message.resourceIDs[i] = $util.newBuffer($util.base64.length(object.resourceIDs[i])), 0);
                        else if (object.resourceIDs[i].length)
                            message.resourceIDs[i] = object.resourceIDs[i];
                }
                if (object.duration != null)
                    if ($util.Long)
                        (message.duration = $util.Long.fromValue(object.duration)).unsigned = false;
                    else if (typeof object.duration === "string")
                        message.duration = parseInt(object.duration, 10);
                    else if (typeof object.duration === "number")
                        message.duration = object.duration;
                    else if (typeof object.duration === "object")
                        message.duration = new $util.LongBits(object.duration.low >>> 0, object.duration.high >>> 0).toNumber();
                return message;
            };
    
            /**
             * Creates a plain object from a ReserveFundTx message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.ReserveFundTx
             * @static
             * @param {serialization.ReserveFundTx} message ReserveFundTx
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ReserveFundTx.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.arrays || options.defaults) {
                    object.collateral = [];
                    object.resourceIDs = [];
                }
                if (options.defaults) {
                    if ($util.Long) {
                        var long = new $util.Long(0, 0, false);
                        object.gas = options.longs === String ? long.toString() : options.longs === Number ? long.toNumber() : long;
                    } else
                        object.gas = options.longs === String ? "0" : 0;
                    object.fee = null;
                    object.source = null;
                    if ($util.Long) {
                        var long = new $util.Long(0, 0, false);
                        object.duration = options.longs === String ? long.toString() : options.longs === Number ? long.toNumber() : long;
                    } else
                        object.duration = options.longs === String ? "0" : 0;
                }
                if (message.gas != null && message.hasOwnProperty("gas"))
                    if (typeof message.gas === "number")
                        object.gas = options.longs === String ? String(message.gas) : message.gas;
                    else
                        object.gas = options.longs === String ? $util.Long.prototype.toString.call(message.gas) : options.longs === Number ? new $util.LongBits(message.gas.low >>> 0, message.gas.high >>> 0).toNumber() : message.gas;
                if (message.fee != null && message.hasOwnProperty("fee"))
                    object.fee = $root.serialization.Coin.toObject(message.fee, options);
                if (message.source != null && message.hasOwnProperty("source"))
                    object.source = $root.serialization.TxInput.toObject(message.source, options);
                if (message.collateral && message.collateral.length) {
                    object.collateral = [];
                    for (var j = 0; j < message.collateral.length; ++j)
                        object.collateral[j] = $root.serialization.Coin.toObject(message.collateral[j], options);
                }
                if (message.resourceIDs && message.resourceIDs.length) {
                    object.resourceIDs = [];
                    for (var j = 0; j < message.resourceIDs.length; ++j)
                        object.resourceIDs[j] = options.bytes === String ? $util.base64.encode(message.resourceIDs[j], 0, message.resourceIDs[j].length) : options.bytes === Array ? Array.prototype.slice.call(message.resourceIDs[j]) : message.resourceIDs[j];
                }
                if (message.duration != null && message.hasOwnProperty("duration"))
                    if (typeof message.duration === "number")
                        object.duration = options.longs === String ? String(message.duration) : message.duration;
                    else
                        object.duration = options.longs === String ? $util.Long.prototype.toString.call(message.duration) : options.longs === Number ? new $util.LongBits(message.duration.low >>> 0, message.duration.high >>> 0).toNumber() : message.duration;
                return object;
            };
    
            /**
             * Converts this ReserveFundTx to JSON.
             * @function toJSON
             * @memberof serialization.ReserveFundTx
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ReserveFundTx.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return ReserveFundTx;
        })();
    
        serialization.ReleaseFundTx = (function() {
    
            /**
             * Properties of a ReleaseFundTx.
             * @memberof serialization
             * @interface IReleaseFundTx
             * @property {number|Long|null} [gas] ReleaseFundTx gas
             * @property {serialization.ICoin|null} [fee] ReleaseFundTx fee
             * @property {serialization.ITxInput|null} [source] ReleaseFundTx source
             * @property {number|Long|null} [reserveSequence] ReleaseFundTx reserveSequence
             */
    
            /**
             * Constructs a new ReleaseFundTx.
             * @memberof serialization
             * @classdesc Represents a ReleaseFundTx.
             * @implements IReleaseFundTx
             * @constructor
             * @param {serialization.IReleaseFundTx=} [properties] Properties to set
             */
            function ReleaseFundTx(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * ReleaseFundTx gas.
             * @member {number|Long} gas
             * @memberof serialization.ReleaseFundTx
             * @instance
             */
            ReleaseFundTx.prototype.gas = $util.Long ? $util.Long.fromBits(0,0,false) : 0;
    
            /**
             * ReleaseFundTx fee.
             * @member {serialization.ICoin|null|undefined} fee
             * @memberof serialization.ReleaseFundTx
             * @instance
             */
            ReleaseFundTx.prototype.fee = null;
    
            /**
             * ReleaseFundTx source.
             * @member {serialization.ITxInput|null|undefined} source
             * @memberof serialization.ReleaseFundTx
             * @instance
             */
            ReleaseFundTx.prototype.source = null;
    
            /**
             * ReleaseFundTx reserveSequence.
             * @member {number|Long} reserveSequence
             * @memberof serialization.ReleaseFundTx
             * @instance
             */
            ReleaseFundTx.prototype.reserveSequence = $util.Long ? $util.Long.fromBits(0,0,false) : 0;
    
            /**
             * Creates a new ReleaseFundTx instance using the specified properties.
             * @function create
             * @memberof serialization.ReleaseFundTx
             * @static
             * @param {serialization.IReleaseFundTx=} [properties] Properties to set
             * @returns {serialization.ReleaseFundTx} ReleaseFundTx instance
             */
            ReleaseFundTx.create = function create(properties) {
                return new ReleaseFundTx(properties);
            };
    
            /**
             * Encodes the specified ReleaseFundTx message. Does not implicitly {@link serialization.ReleaseFundTx.verify|verify} messages.
             * @function encode
             * @memberof serialization.ReleaseFundTx
             * @static
             * @param {serialization.IReleaseFundTx} message ReleaseFundTx message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ReleaseFundTx.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.gas != null && message.hasOwnProperty("gas"))
                    writer.uint32(/* id 1, wireType 0 =*/8).int64(message.gas);
                if (message.fee != null && message.hasOwnProperty("fee"))
                    $root.serialization.Coin.encode(message.fee, writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
                if (message.source != null && message.hasOwnProperty("source"))
                    $root.serialization.TxInput.encode(message.source, writer.uint32(/* id 3, wireType 2 =*/26).fork()).ldelim();
                if (message.reserveSequence != null && message.hasOwnProperty("reserveSequence"))
                    writer.uint32(/* id 4, wireType 0 =*/32).int64(message.reserveSequence);
                return writer;
            };
    
            /**
             * Encodes the specified ReleaseFundTx message, length delimited. Does not implicitly {@link serialization.ReleaseFundTx.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.ReleaseFundTx
             * @static
             * @param {serialization.IReleaseFundTx} message ReleaseFundTx message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ReleaseFundTx.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes a ReleaseFundTx message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.ReleaseFundTx
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.ReleaseFundTx} ReleaseFundTx
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ReleaseFundTx.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.ReleaseFundTx();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.gas = reader.int64();
                        break;
                    case 2:
                        message.fee = $root.serialization.Coin.decode(reader, reader.uint32());
                        break;
                    case 3:
                        message.source = $root.serialization.TxInput.decode(reader, reader.uint32());
                        break;
                    case 4:
                        message.reserveSequence = reader.int64();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes a ReleaseFundTx message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.ReleaseFundTx
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.ReleaseFundTx} ReleaseFundTx
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ReleaseFundTx.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies a ReleaseFundTx message.
             * @function verify
             * @memberof serialization.ReleaseFundTx
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ReleaseFundTx.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.gas != null && message.hasOwnProperty("gas"))
                    if (!$util.isInteger(message.gas) && !(message.gas && $util.isInteger(message.gas.low) && $util.isInteger(message.gas.high)))
                        return "gas: integer|Long expected";
                if (message.fee != null && message.hasOwnProperty("fee")) {
                    var error = $root.serialization.Coin.verify(message.fee);
                    if (error)
                        return "fee." + error;
                }
                if (message.source != null && message.hasOwnProperty("source")) {
                    var error = $root.serialization.TxInput.verify(message.source);
                    if (error)
                        return "source." + error;
                }
                if (message.reserveSequence != null && message.hasOwnProperty("reserveSequence"))
                    if (!$util.isInteger(message.reserveSequence) && !(message.reserveSequence && $util.isInteger(message.reserveSequence.low) && $util.isInteger(message.reserveSequence.high)))
                        return "reserveSequence: integer|Long expected";
                return null;
            };
    
            /**
             * Creates a ReleaseFundTx message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.ReleaseFundTx
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.ReleaseFundTx} ReleaseFundTx
             */
            ReleaseFundTx.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.ReleaseFundTx)
                    return object;
                var message = new $root.serialization.ReleaseFundTx();
                if (object.gas != null)
                    if ($util.Long)
                        (message.gas = $util.Long.fromValue(object.gas)).unsigned = false;
                    else if (typeof object.gas === "string")
                        message.gas = parseInt(object.gas, 10);
                    else if (typeof object.gas === "number")
                        message.gas = object.gas;
                    else if (typeof object.gas === "object")
                        message.gas = new $util.LongBits(object.gas.low >>> 0, object.gas.high >>> 0).toNumber();
                if (object.fee != null) {
                    if (typeof object.fee !== "object")
                        throw TypeError(".serialization.ReleaseFundTx.fee: object expected");
                    message.fee = $root.serialization.Coin.fromObject(object.fee);
                }
                if (object.source != null) {
                    if (typeof object.source !== "object")
                        throw TypeError(".serialization.ReleaseFundTx.source: object expected");
                    message.source = $root.serialization.TxInput.fromObject(object.source);
                }
                if (object.reserveSequence != null)
                    if ($util.Long)
                        (message.reserveSequence = $util.Long.fromValue(object.reserveSequence)).unsigned = false;
                    else if (typeof object.reserveSequence === "string")
                        message.reserveSequence = parseInt(object.reserveSequence, 10);
                    else if (typeof object.reserveSequence === "number")
                        message.reserveSequence = object.reserveSequence;
                    else if (typeof object.reserveSequence === "object")
                        message.reserveSequence = new $util.LongBits(object.reserveSequence.low >>> 0, object.reserveSequence.high >>> 0).toNumber();
                return message;
            };
    
            /**
             * Creates a plain object from a ReleaseFundTx message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.ReleaseFundTx
             * @static
             * @param {serialization.ReleaseFundTx} message ReleaseFundTx
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ReleaseFundTx.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults) {
                    if ($util.Long) {
                        var long = new $util.Long(0, 0, false);
                        object.gas = options.longs === String ? long.toString() : options.longs === Number ? long.toNumber() : long;
                    } else
                        object.gas = options.longs === String ? "0" : 0;
                    object.fee = null;
                    object.source = null;
                    if ($util.Long) {
                        var long = new $util.Long(0, 0, false);
                        object.reserveSequence = options.longs === String ? long.toString() : options.longs === Number ? long.toNumber() : long;
                    } else
                        object.reserveSequence = options.longs === String ? "0" : 0;
                }
                if (message.gas != null && message.hasOwnProperty("gas"))
                    if (typeof message.gas === "number")
                        object.gas = options.longs === String ? String(message.gas) : message.gas;
                    else
                        object.gas = options.longs === String ? $util.Long.prototype.toString.call(message.gas) : options.longs === Number ? new $util.LongBits(message.gas.low >>> 0, message.gas.high >>> 0).toNumber() : message.gas;
                if (message.fee != null && message.hasOwnProperty("fee"))
                    object.fee = $root.serialization.Coin.toObject(message.fee, options);
                if (message.source != null && message.hasOwnProperty("source"))
                    object.source = $root.serialization.TxInput.toObject(message.source, options);
                if (message.reserveSequence != null && message.hasOwnProperty("reserveSequence"))
                    if (typeof message.reserveSequence === "number")
                        object.reserveSequence = options.longs === String ? String(message.reserveSequence) : message.reserveSequence;
                    else
                        object.reserveSequence = options.longs === String ? $util.Long.prototype.toString.call(message.reserveSequence) : options.longs === Number ? new $util.LongBits(message.reserveSequence.low >>> 0, message.reserveSequence.high >>> 0).toNumber() : message.reserveSequence;
                return object;
            };
    
            /**
             * Converts this ReleaseFundTx to JSON.
             * @function toJSON
             * @memberof serialization.ReleaseFundTx
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ReleaseFundTx.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return ReleaseFundTx;
        })();
    
        serialization.ServicePaymentTx = (function() {
    
            /**
             * Properties of a ServicePaymentTx.
             * @memberof serialization
             * @interface IServicePaymentTx
             * @property {number|Long|null} [gas] ServicePaymentTx gas
             * @property {serialization.ICoin|null} [fee] ServicePaymentTx fee
             * @property {serialization.ITxInput|null} [source] ServicePaymentTx source
             * @property {serialization.ITxInput|null} [target] ServicePaymentTx target
             * @property {number|Long|null} [PaymentSequence] ServicePaymentTx PaymentSequence
             * @property {number|Long|null} [ReserveSequence] ServicePaymentTx ReserveSequence
             * @property {Uint8Array|null} [ResourceID] ServicePaymentTx ResourceID
             */
    
            /**
             * Constructs a new ServicePaymentTx.
             * @memberof serialization
             * @classdesc Represents a ServicePaymentTx.
             * @implements IServicePaymentTx
             * @constructor
             * @param {serialization.IServicePaymentTx=} [properties] Properties to set
             */
            function ServicePaymentTx(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * ServicePaymentTx gas.
             * @member {number|Long} gas
             * @memberof serialization.ServicePaymentTx
             * @instance
             */
            ServicePaymentTx.prototype.gas = $util.Long ? $util.Long.fromBits(0,0,false) : 0;
    
            /**
             * ServicePaymentTx fee.
             * @member {serialization.ICoin|null|undefined} fee
             * @memberof serialization.ServicePaymentTx
             * @instance
             */
            ServicePaymentTx.prototype.fee = null;
    
            /**
             * ServicePaymentTx source.
             * @member {serialization.ITxInput|null|undefined} source
             * @memberof serialization.ServicePaymentTx
             * @instance
             */
            ServicePaymentTx.prototype.source = null;
    
            /**
             * ServicePaymentTx target.
             * @member {serialization.ITxInput|null|undefined} target
             * @memberof serialization.ServicePaymentTx
             * @instance
             */
            ServicePaymentTx.prototype.target = null;
    
            /**
             * ServicePaymentTx PaymentSequence.
             * @member {number|Long} PaymentSequence
             * @memberof serialization.ServicePaymentTx
             * @instance
             */
            ServicePaymentTx.prototype.PaymentSequence = $util.Long ? $util.Long.fromBits(0,0,false) : 0;
    
            /**
             * ServicePaymentTx ReserveSequence.
             * @member {number|Long} ReserveSequence
             * @memberof serialization.ServicePaymentTx
             * @instance
             */
            ServicePaymentTx.prototype.ReserveSequence = $util.Long ? $util.Long.fromBits(0,0,false) : 0;
    
            /**
             * ServicePaymentTx ResourceID.
             * @member {Uint8Array} ResourceID
             * @memberof serialization.ServicePaymentTx
             * @instance
             */
            ServicePaymentTx.prototype.ResourceID = $util.newBuffer([]);
    
            /**
             * Creates a new ServicePaymentTx instance using the specified properties.
             * @function create
             * @memberof serialization.ServicePaymentTx
             * @static
             * @param {serialization.IServicePaymentTx=} [properties] Properties to set
             * @returns {serialization.ServicePaymentTx} ServicePaymentTx instance
             */
            ServicePaymentTx.create = function create(properties) {
                return new ServicePaymentTx(properties);
            };
    
            /**
             * Encodes the specified ServicePaymentTx message. Does not implicitly {@link serialization.ServicePaymentTx.verify|verify} messages.
             * @function encode
             * @memberof serialization.ServicePaymentTx
             * @static
             * @param {serialization.IServicePaymentTx} message ServicePaymentTx message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ServicePaymentTx.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.gas != null && message.hasOwnProperty("gas"))
                    writer.uint32(/* id 1, wireType 0 =*/8).int64(message.gas);
                if (message.fee != null && message.hasOwnProperty("fee"))
                    $root.serialization.Coin.encode(message.fee, writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
                if (message.source != null && message.hasOwnProperty("source"))
                    $root.serialization.TxInput.encode(message.source, writer.uint32(/* id 3, wireType 2 =*/26).fork()).ldelim();
                if (message.target != null && message.hasOwnProperty("target"))
                    $root.serialization.TxInput.encode(message.target, writer.uint32(/* id 4, wireType 2 =*/34).fork()).ldelim();
                if (message.PaymentSequence != null && message.hasOwnProperty("PaymentSequence"))
                    writer.uint32(/* id 5, wireType 0 =*/40).int64(message.PaymentSequence);
                if (message.ReserveSequence != null && message.hasOwnProperty("ReserveSequence"))
                    writer.uint32(/* id 6, wireType 0 =*/48).int64(message.ReserveSequence);
                if (message.ResourceID != null && message.hasOwnProperty("ResourceID"))
                    writer.uint32(/* id 7, wireType 2 =*/58).bytes(message.ResourceID);
                return writer;
            };
    
            /**
             * Encodes the specified ServicePaymentTx message, length delimited. Does not implicitly {@link serialization.ServicePaymentTx.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.ServicePaymentTx
             * @static
             * @param {serialization.IServicePaymentTx} message ServicePaymentTx message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            ServicePaymentTx.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes a ServicePaymentTx message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.ServicePaymentTx
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.ServicePaymentTx} ServicePaymentTx
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ServicePaymentTx.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.ServicePaymentTx();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.gas = reader.int64();
                        break;
                    case 2:
                        message.fee = $root.serialization.Coin.decode(reader, reader.uint32());
                        break;
                    case 3:
                        message.source = $root.serialization.TxInput.decode(reader, reader.uint32());
                        break;
                    case 4:
                        message.target = $root.serialization.TxInput.decode(reader, reader.uint32());
                        break;
                    case 5:
                        message.PaymentSequence = reader.int64();
                        break;
                    case 6:
                        message.ReserveSequence = reader.int64();
                        break;
                    case 7:
                        message.ResourceID = reader.bytes();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes a ServicePaymentTx message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.ServicePaymentTx
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.ServicePaymentTx} ServicePaymentTx
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            ServicePaymentTx.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies a ServicePaymentTx message.
             * @function verify
             * @memberof serialization.ServicePaymentTx
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            ServicePaymentTx.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.gas != null && message.hasOwnProperty("gas"))
                    if (!$util.isInteger(message.gas) && !(message.gas && $util.isInteger(message.gas.low) && $util.isInteger(message.gas.high)))
                        return "gas: integer|Long expected";
                if (message.fee != null && message.hasOwnProperty("fee")) {
                    var error = $root.serialization.Coin.verify(message.fee);
                    if (error)
                        return "fee." + error;
                }
                if (message.source != null && message.hasOwnProperty("source")) {
                    var error = $root.serialization.TxInput.verify(message.source);
                    if (error)
                        return "source." + error;
                }
                if (message.target != null && message.hasOwnProperty("target")) {
                    var error = $root.serialization.TxInput.verify(message.target);
                    if (error)
                        return "target." + error;
                }
                if (message.PaymentSequence != null && message.hasOwnProperty("PaymentSequence"))
                    if (!$util.isInteger(message.PaymentSequence) && !(message.PaymentSequence && $util.isInteger(message.PaymentSequence.low) && $util.isInteger(message.PaymentSequence.high)))
                        return "PaymentSequence: integer|Long expected";
                if (message.ReserveSequence != null && message.hasOwnProperty("ReserveSequence"))
                    if (!$util.isInteger(message.ReserveSequence) && !(message.ReserveSequence && $util.isInteger(message.ReserveSequence.low) && $util.isInteger(message.ReserveSequence.high)))
                        return "ReserveSequence: integer|Long expected";
                if (message.ResourceID != null && message.hasOwnProperty("ResourceID"))
                    if (!(message.ResourceID && typeof message.ResourceID.length === "number" || $util.isString(message.ResourceID)))
                        return "ResourceID: buffer expected";
                return null;
            };
    
            /**
             * Creates a ServicePaymentTx message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.ServicePaymentTx
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.ServicePaymentTx} ServicePaymentTx
             */
            ServicePaymentTx.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.ServicePaymentTx)
                    return object;
                var message = new $root.serialization.ServicePaymentTx();
                if (object.gas != null)
                    if ($util.Long)
                        (message.gas = $util.Long.fromValue(object.gas)).unsigned = false;
                    else if (typeof object.gas === "string")
                        message.gas = parseInt(object.gas, 10);
                    else if (typeof object.gas === "number")
                        message.gas = object.gas;
                    else if (typeof object.gas === "object")
                        message.gas = new $util.LongBits(object.gas.low >>> 0, object.gas.high >>> 0).toNumber();
                if (object.fee != null) {
                    if (typeof object.fee !== "object")
                        throw TypeError(".serialization.ServicePaymentTx.fee: object expected");
                    message.fee = $root.serialization.Coin.fromObject(object.fee);
                }
                if (object.source != null) {
                    if (typeof object.source !== "object")
                        throw TypeError(".serialization.ServicePaymentTx.source: object expected");
                    message.source = $root.serialization.TxInput.fromObject(object.source);
                }
                if (object.target != null) {
                    if (typeof object.target !== "object")
                        throw TypeError(".serialization.ServicePaymentTx.target: object expected");
                    message.target = $root.serialization.TxInput.fromObject(object.target);
                }
                if (object.PaymentSequence != null)
                    if ($util.Long)
                        (message.PaymentSequence = $util.Long.fromValue(object.PaymentSequence)).unsigned = false;
                    else if (typeof object.PaymentSequence === "string")
                        message.PaymentSequence = parseInt(object.PaymentSequence, 10);
                    else if (typeof object.PaymentSequence === "number")
                        message.PaymentSequence = object.PaymentSequence;
                    else if (typeof object.PaymentSequence === "object")
                        message.PaymentSequence = new $util.LongBits(object.PaymentSequence.low >>> 0, object.PaymentSequence.high >>> 0).toNumber();
                if (object.ReserveSequence != null)
                    if ($util.Long)
                        (message.ReserveSequence = $util.Long.fromValue(object.ReserveSequence)).unsigned = false;
                    else if (typeof object.ReserveSequence === "string")
                        message.ReserveSequence = parseInt(object.ReserveSequence, 10);
                    else if (typeof object.ReserveSequence === "number")
                        message.ReserveSequence = object.ReserveSequence;
                    else if (typeof object.ReserveSequence === "object")
                        message.ReserveSequence = new $util.LongBits(object.ReserveSequence.low >>> 0, object.ReserveSequence.high >>> 0).toNumber();
                if (object.ResourceID != null)
                    if (typeof object.ResourceID === "string")
                        $util.base64.decode(object.ResourceID, message.ResourceID = $util.newBuffer($util.base64.length(object.ResourceID)), 0);
                    else if (object.ResourceID.length)
                        message.ResourceID = object.ResourceID;
                return message;
            };
    
            /**
             * Creates a plain object from a ServicePaymentTx message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.ServicePaymentTx
             * @static
             * @param {serialization.ServicePaymentTx} message ServicePaymentTx
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            ServicePaymentTx.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults) {
                    if ($util.Long) {
                        var long = new $util.Long(0, 0, false);
                        object.gas = options.longs === String ? long.toString() : options.longs === Number ? long.toNumber() : long;
                    } else
                        object.gas = options.longs === String ? "0" : 0;
                    object.fee = null;
                    object.source = null;
                    object.target = null;
                    if ($util.Long) {
                        var long = new $util.Long(0, 0, false);
                        object.PaymentSequence = options.longs === String ? long.toString() : options.longs === Number ? long.toNumber() : long;
                    } else
                        object.PaymentSequence = options.longs === String ? "0" : 0;
                    if ($util.Long) {
                        var long = new $util.Long(0, 0, false);
                        object.ReserveSequence = options.longs === String ? long.toString() : options.longs === Number ? long.toNumber() : long;
                    } else
                        object.ReserveSequence = options.longs === String ? "0" : 0;
                    object.ResourceID = options.bytes === String ? "" : [];
                }
                if (message.gas != null && message.hasOwnProperty("gas"))
                    if (typeof message.gas === "number")
                        object.gas = options.longs === String ? String(message.gas) : message.gas;
                    else
                        object.gas = options.longs === String ? $util.Long.prototype.toString.call(message.gas) : options.longs === Number ? new $util.LongBits(message.gas.low >>> 0, message.gas.high >>> 0).toNumber() : message.gas;
                if (message.fee != null && message.hasOwnProperty("fee"))
                    object.fee = $root.serialization.Coin.toObject(message.fee, options);
                if (message.source != null && message.hasOwnProperty("source"))
                    object.source = $root.serialization.TxInput.toObject(message.source, options);
                if (message.target != null && message.hasOwnProperty("target"))
                    object.target = $root.serialization.TxInput.toObject(message.target, options);
                if (message.PaymentSequence != null && message.hasOwnProperty("PaymentSequence"))
                    if (typeof message.PaymentSequence === "number")
                        object.PaymentSequence = options.longs === String ? String(message.PaymentSequence) : message.PaymentSequence;
                    else
                        object.PaymentSequence = options.longs === String ? $util.Long.prototype.toString.call(message.PaymentSequence) : options.longs === Number ? new $util.LongBits(message.PaymentSequence.low >>> 0, message.PaymentSequence.high >>> 0).toNumber() : message.PaymentSequence;
                if (message.ReserveSequence != null && message.hasOwnProperty("ReserveSequence"))
                    if (typeof message.ReserveSequence === "number")
                        object.ReserveSequence = options.longs === String ? String(message.ReserveSequence) : message.ReserveSequence;
                    else
                        object.ReserveSequence = options.longs === String ? $util.Long.prototype.toString.call(message.ReserveSequence) : options.longs === Number ? new $util.LongBits(message.ReserveSequence.low >>> 0, message.ReserveSequence.high >>> 0).toNumber() : message.ReserveSequence;
                if (message.ResourceID != null && message.hasOwnProperty("ResourceID"))
                    object.ResourceID = options.bytes === String ? $util.base64.encode(message.ResourceID, 0, message.ResourceID.length) : options.bytes === Array ? Array.prototype.slice.call(message.ResourceID) : message.ResourceID;
                return object;
            };
    
            /**
             * Converts this ServicePaymentTx to JSON.
             * @function toJSON
             * @memberof serialization.ServicePaymentTx
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            ServicePaymentTx.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return ServicePaymentTx;
        })();
    
        serialization.SlashTx = (function() {
    
            /**
             * Properties of a SlashTx.
             * @memberof serialization
             * @interface ISlashTx
             * @property {serialization.ITxInput|null} [proposer] SlashTx proposer
             * @property {Uint8Array|null} [slashedAddress] SlashTx slashedAddress
             * @property {number|Long|null} [reserveSequence] SlashTx reserveSequence
             * @property {Uint8Array|null} [slashProof] SlashTx slashProof
             */
    
            /**
             * Constructs a new SlashTx.
             * @memberof serialization
             * @classdesc Represents a SlashTx.
             * @implements ISlashTx
             * @constructor
             * @param {serialization.ISlashTx=} [properties] Properties to set
             */
            function SlashTx(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * SlashTx proposer.
             * @member {serialization.ITxInput|null|undefined} proposer
             * @memberof serialization.SlashTx
             * @instance
             */
            SlashTx.prototype.proposer = null;
    
            /**
             * SlashTx slashedAddress.
             * @member {Uint8Array} slashedAddress
             * @memberof serialization.SlashTx
             * @instance
             */
            SlashTx.prototype.slashedAddress = $util.newBuffer([]);
    
            /**
             * SlashTx reserveSequence.
             * @member {number|Long} reserveSequence
             * @memberof serialization.SlashTx
             * @instance
             */
            SlashTx.prototype.reserveSequence = $util.Long ? $util.Long.fromBits(0,0,false) : 0;
    
            /**
             * SlashTx slashProof.
             * @member {Uint8Array} slashProof
             * @memberof serialization.SlashTx
             * @instance
             */
            SlashTx.prototype.slashProof = $util.newBuffer([]);
    
            /**
             * Creates a new SlashTx instance using the specified properties.
             * @function create
             * @memberof serialization.SlashTx
             * @static
             * @param {serialization.ISlashTx=} [properties] Properties to set
             * @returns {serialization.SlashTx} SlashTx instance
             */
            SlashTx.create = function create(properties) {
                return new SlashTx(properties);
            };
    
            /**
             * Encodes the specified SlashTx message. Does not implicitly {@link serialization.SlashTx.verify|verify} messages.
             * @function encode
             * @memberof serialization.SlashTx
             * @static
             * @param {serialization.ISlashTx} message SlashTx message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            SlashTx.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.proposer != null && message.hasOwnProperty("proposer"))
                    $root.serialization.TxInput.encode(message.proposer, writer.uint32(/* id 1, wireType 2 =*/10).fork()).ldelim();
                if (message.slashedAddress != null && message.hasOwnProperty("slashedAddress"))
                    writer.uint32(/* id 2, wireType 2 =*/18).bytes(message.slashedAddress);
                if (message.reserveSequence != null && message.hasOwnProperty("reserveSequence"))
                    writer.uint32(/* id 3, wireType 0 =*/24).int64(message.reserveSequence);
                if (message.slashProof != null && message.hasOwnProperty("slashProof"))
                    writer.uint32(/* id 4, wireType 2 =*/34).bytes(message.slashProof);
                return writer;
            };
    
            /**
             * Encodes the specified SlashTx message, length delimited. Does not implicitly {@link serialization.SlashTx.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.SlashTx
             * @static
             * @param {serialization.ISlashTx} message SlashTx message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            SlashTx.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes a SlashTx message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.SlashTx
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.SlashTx} SlashTx
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            SlashTx.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.SlashTx();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.proposer = $root.serialization.TxInput.decode(reader, reader.uint32());
                        break;
                    case 2:
                        message.slashedAddress = reader.bytes();
                        break;
                    case 3:
                        message.reserveSequence = reader.int64();
                        break;
                    case 4:
                        message.slashProof = reader.bytes();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes a SlashTx message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.SlashTx
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.SlashTx} SlashTx
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            SlashTx.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies a SlashTx message.
             * @function verify
             * @memberof serialization.SlashTx
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            SlashTx.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.proposer != null && message.hasOwnProperty("proposer")) {
                    var error = $root.serialization.TxInput.verify(message.proposer);
                    if (error)
                        return "proposer." + error;
                }
                if (message.slashedAddress != null && message.hasOwnProperty("slashedAddress"))
                    if (!(message.slashedAddress && typeof message.slashedAddress.length === "number" || $util.isString(message.slashedAddress)))
                        return "slashedAddress: buffer expected";
                if (message.reserveSequence != null && message.hasOwnProperty("reserveSequence"))
                    if (!$util.isInteger(message.reserveSequence) && !(message.reserveSequence && $util.isInteger(message.reserveSequence.low) && $util.isInteger(message.reserveSequence.high)))
                        return "reserveSequence: integer|Long expected";
                if (message.slashProof != null && message.hasOwnProperty("slashProof"))
                    if (!(message.slashProof && typeof message.slashProof.length === "number" || $util.isString(message.slashProof)))
                        return "slashProof: buffer expected";
                return null;
            };
    
            /**
             * Creates a SlashTx message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.SlashTx
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.SlashTx} SlashTx
             */
            SlashTx.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.SlashTx)
                    return object;
                var message = new $root.serialization.SlashTx();
                if (object.proposer != null) {
                    if (typeof object.proposer !== "object")
                        throw TypeError(".serialization.SlashTx.proposer: object expected");
                    message.proposer = $root.serialization.TxInput.fromObject(object.proposer);
                }
                if (object.slashedAddress != null)
                    if (typeof object.slashedAddress === "string")
                        $util.base64.decode(object.slashedAddress, message.slashedAddress = $util.newBuffer($util.base64.length(object.slashedAddress)), 0);
                    else if (object.slashedAddress.length)
                        message.slashedAddress = object.slashedAddress;
                if (object.reserveSequence != null)
                    if ($util.Long)
                        (message.reserveSequence = $util.Long.fromValue(object.reserveSequence)).unsigned = false;
                    else if (typeof object.reserveSequence === "string")
                        message.reserveSequence = parseInt(object.reserveSequence, 10);
                    else if (typeof object.reserveSequence === "number")
                        message.reserveSequence = object.reserveSequence;
                    else if (typeof object.reserveSequence === "object")
                        message.reserveSequence = new $util.LongBits(object.reserveSequence.low >>> 0, object.reserveSequence.high >>> 0).toNumber();
                if (object.slashProof != null)
                    if (typeof object.slashProof === "string")
                        $util.base64.decode(object.slashProof, message.slashProof = $util.newBuffer($util.base64.length(object.slashProof)), 0);
                    else if (object.slashProof.length)
                        message.slashProof = object.slashProof;
                return message;
            };
    
            /**
             * Creates a plain object from a SlashTx message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.SlashTx
             * @static
             * @param {serialization.SlashTx} message SlashTx
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            SlashTx.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults) {
                    object.proposer = null;
                    object.slashedAddress = options.bytes === String ? "" : [];
                    if ($util.Long) {
                        var long = new $util.Long(0, 0, false);
                        object.reserveSequence = options.longs === String ? long.toString() : options.longs === Number ? long.toNumber() : long;
                    } else
                        object.reserveSequence = options.longs === String ? "0" : 0;
                    object.slashProof = options.bytes === String ? "" : [];
                }
                if (message.proposer != null && message.hasOwnProperty("proposer"))
                    object.proposer = $root.serialization.TxInput.toObject(message.proposer, options);
                if (message.slashedAddress != null && message.hasOwnProperty("slashedAddress"))
                    object.slashedAddress = options.bytes === String ? $util.base64.encode(message.slashedAddress, 0, message.slashedAddress.length) : options.bytes === Array ? Array.prototype.slice.call(message.slashedAddress) : message.slashedAddress;
                if (message.reserveSequence != null && message.hasOwnProperty("reserveSequence"))
                    if (typeof message.reserveSequence === "number")
                        object.reserveSequence = options.longs === String ? String(message.reserveSequence) : message.reserveSequence;
                    else
                        object.reserveSequence = options.longs === String ? $util.Long.prototype.toString.call(message.reserveSequence) : options.longs === Number ? new $util.LongBits(message.reserveSequence.low >>> 0, message.reserveSequence.high >>> 0).toNumber() : message.reserveSequence;
                if (message.slashProof != null && message.hasOwnProperty("slashProof"))
                    object.slashProof = options.bytes === String ? $util.base64.encode(message.slashProof, 0, message.slashProof.length) : options.bytes === Array ? Array.prototype.slice.call(message.slashProof) : message.slashProof;
                return object;
            };
    
            /**
             * Converts this SlashTx to JSON.
             * @function toJSON
             * @memberof serialization.SlashTx
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            SlashTx.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return SlashTx;
        })();
    
        serialization.Split = (function() {
    
            /**
             * Properties of a Split.
             * @memberof serialization
             * @interface ISplit
             * @property {Uint8Array|null} [address] Split address
             * @property {number|Long|null} [percentage] Split percentage
             */
    
            /**
             * Constructs a new Split.
             * @memberof serialization
             * @classdesc Represents a Split.
             * @implements ISplit
             * @constructor
             * @param {serialization.ISplit=} [properties] Properties to set
             */
            function Split(properties) {
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * Split address.
             * @member {Uint8Array} address
             * @memberof serialization.Split
             * @instance
             */
            Split.prototype.address = $util.newBuffer([]);
    
            /**
             * Split percentage.
             * @member {number|Long} percentage
             * @memberof serialization.Split
             * @instance
             */
            Split.prototype.percentage = $util.Long ? $util.Long.fromBits(0,0,false) : 0;
    
            /**
             * Creates a new Split instance using the specified properties.
             * @function create
             * @memberof serialization.Split
             * @static
             * @param {serialization.ISplit=} [properties] Properties to set
             * @returns {serialization.Split} Split instance
             */
            Split.create = function create(properties) {
                return new Split(properties);
            };
    
            /**
             * Encodes the specified Split message. Does not implicitly {@link serialization.Split.verify|verify} messages.
             * @function encode
             * @memberof serialization.Split
             * @static
             * @param {serialization.ISplit} message Split message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Split.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.address != null && message.hasOwnProperty("address"))
                    writer.uint32(/* id 1, wireType 2 =*/10).bytes(message.address);
                if (message.percentage != null && message.hasOwnProperty("percentage"))
                    writer.uint32(/* id 2, wireType 0 =*/16).int64(message.percentage);
                return writer;
            };
    
            /**
             * Encodes the specified Split message, length delimited. Does not implicitly {@link serialization.Split.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.Split
             * @static
             * @param {serialization.ISplit} message Split message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            Split.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes a Split message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.Split
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.Split} Split
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Split.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.Split();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.address = reader.bytes();
                        break;
                    case 2:
                        message.percentage = reader.int64();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes a Split message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.Split
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.Split} Split
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            Split.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies a Split message.
             * @function verify
             * @memberof serialization.Split
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            Split.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.address != null && message.hasOwnProperty("address"))
                    if (!(message.address && typeof message.address.length === "number" || $util.isString(message.address)))
                        return "address: buffer expected";
                if (message.percentage != null && message.hasOwnProperty("percentage"))
                    if (!$util.isInteger(message.percentage) && !(message.percentage && $util.isInteger(message.percentage.low) && $util.isInteger(message.percentage.high)))
                        return "percentage: integer|Long expected";
                return null;
            };
    
            /**
             * Creates a Split message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.Split
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.Split} Split
             */
            Split.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.Split)
                    return object;
                var message = new $root.serialization.Split();
                if (object.address != null)
                    if (typeof object.address === "string")
                        $util.base64.decode(object.address, message.address = $util.newBuffer($util.base64.length(object.address)), 0);
                    else if (object.address.length)
                        message.address = object.address;
                if (object.percentage != null)
                    if ($util.Long)
                        (message.percentage = $util.Long.fromValue(object.percentage)).unsigned = false;
                    else if (typeof object.percentage === "string")
                        message.percentage = parseInt(object.percentage, 10);
                    else if (typeof object.percentage === "number")
                        message.percentage = object.percentage;
                    else if (typeof object.percentage === "object")
                        message.percentage = new $util.LongBits(object.percentage.low >>> 0, object.percentage.high >>> 0).toNumber();
                return message;
            };
    
            /**
             * Creates a plain object from a Split message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.Split
             * @static
             * @param {serialization.Split} message Split
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            Split.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.defaults) {
                    object.address = options.bytes === String ? "" : [];
                    if ($util.Long) {
                        var long = new $util.Long(0, 0, false);
                        object.percentage = options.longs === String ? long.toString() : options.longs === Number ? long.toNumber() : long;
                    } else
                        object.percentage = options.longs === String ? "0" : 0;
                }
                if (message.address != null && message.hasOwnProperty("address"))
                    object.address = options.bytes === String ? $util.base64.encode(message.address, 0, message.address.length) : options.bytes === Array ? Array.prototype.slice.call(message.address) : message.address;
                if (message.percentage != null && message.hasOwnProperty("percentage"))
                    if (typeof message.percentage === "number")
                        object.percentage = options.longs === String ? String(message.percentage) : message.percentage;
                    else
                        object.percentage = options.longs === String ? $util.Long.prototype.toString.call(message.percentage) : options.longs === Number ? new $util.LongBits(message.percentage.low >>> 0, message.percentage.high >>> 0).toNumber() : message.percentage;
                return object;
            };
    
            /**
             * Converts this Split to JSON.
             * @function toJSON
             * @memberof serialization.Split
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            Split.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return Split;
        })();
    
        serialization.SplitContract = (function() {
    
            /**
             * Properties of a SplitContract.
             * @memberof serialization
             * @interface ISplitContract
             * @property {Uint8Array|null} [initiatorAddress] SplitContract initiatorAddress
             * @property {Uint8Array|null} [resourceID] SplitContract resourceID
             * @property {Array.<serialization.ISplit>|null} [splits] SplitContract splits
             * @property {number|null} [endBlockHeight] SplitContract endBlockHeight
             */
    
            /**
             * Constructs a new SplitContract.
             * @memberof serialization
             * @classdesc Represents a SplitContract.
             * @implements ISplitContract
             * @constructor
             * @param {serialization.ISplitContract=} [properties] Properties to set
             */
            function SplitContract(properties) {
                this.splits = [];
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * SplitContract initiatorAddress.
             * @member {Uint8Array} initiatorAddress
             * @memberof serialization.SplitContract
             * @instance
             */
            SplitContract.prototype.initiatorAddress = $util.newBuffer([]);
    
            /**
             * SplitContract resourceID.
             * @member {Uint8Array} resourceID
             * @memberof serialization.SplitContract
             * @instance
             */
            SplitContract.prototype.resourceID = $util.newBuffer([]);
    
            /**
             * SplitContract splits.
             * @member {Array.<serialization.ISplit>} splits
             * @memberof serialization.SplitContract
             * @instance
             */
            SplitContract.prototype.splits = $util.emptyArray;
    
            /**
             * SplitContract endBlockHeight.
             * @member {number} endBlockHeight
             * @memberof serialization.SplitContract
             * @instance
             */
            SplitContract.prototype.endBlockHeight = 0;
    
            /**
             * Creates a new SplitContract instance using the specified properties.
             * @function create
             * @memberof serialization.SplitContract
             * @static
             * @param {serialization.ISplitContract=} [properties] Properties to set
             * @returns {serialization.SplitContract} SplitContract instance
             */
            SplitContract.create = function create(properties) {
                return new SplitContract(properties);
            };
    
            /**
             * Encodes the specified SplitContract message. Does not implicitly {@link serialization.SplitContract.verify|verify} messages.
             * @function encode
             * @memberof serialization.SplitContract
             * @static
             * @param {serialization.ISplitContract} message SplitContract message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            SplitContract.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.initiatorAddress != null && message.hasOwnProperty("initiatorAddress"))
                    writer.uint32(/* id 1, wireType 2 =*/10).bytes(message.initiatorAddress);
                if (message.resourceID != null && message.hasOwnProperty("resourceID"))
                    writer.uint32(/* id 2, wireType 2 =*/18).bytes(message.resourceID);
                if (message.splits != null && message.splits.length)
                    for (var i = 0; i < message.splits.length; ++i)
                        $root.serialization.Split.encode(message.splits[i], writer.uint32(/* id 3, wireType 2 =*/26).fork()).ldelim();
                if (message.endBlockHeight != null && message.hasOwnProperty("endBlockHeight"))
                    writer.uint32(/* id 4, wireType 0 =*/32).int32(message.endBlockHeight);
                return writer;
            };
    
            /**
             * Encodes the specified SplitContract message, length delimited. Does not implicitly {@link serialization.SplitContract.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.SplitContract
             * @static
             * @param {serialization.ISplitContract} message SplitContract message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            SplitContract.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes a SplitContract message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.SplitContract
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.SplitContract} SplitContract
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            SplitContract.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.SplitContract();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.initiatorAddress = reader.bytes();
                        break;
                    case 2:
                        message.resourceID = reader.bytes();
                        break;
                    case 3:
                        if (!(message.splits && message.splits.length))
                            message.splits = [];
                        message.splits.push($root.serialization.Split.decode(reader, reader.uint32()));
                        break;
                    case 4:
                        message.endBlockHeight = reader.int32();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes a SplitContract message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.SplitContract
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.SplitContract} SplitContract
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            SplitContract.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies a SplitContract message.
             * @function verify
             * @memberof serialization.SplitContract
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            SplitContract.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.initiatorAddress != null && message.hasOwnProperty("initiatorAddress"))
                    if (!(message.initiatorAddress && typeof message.initiatorAddress.length === "number" || $util.isString(message.initiatorAddress)))
                        return "initiatorAddress: buffer expected";
                if (message.resourceID != null && message.hasOwnProperty("resourceID"))
                    if (!(message.resourceID && typeof message.resourceID.length === "number" || $util.isString(message.resourceID)))
                        return "resourceID: buffer expected";
                if (message.splits != null && message.hasOwnProperty("splits")) {
                    if (!Array.isArray(message.splits))
                        return "splits: array expected";
                    for (var i = 0; i < message.splits.length; ++i) {
                        var error = $root.serialization.Split.verify(message.splits[i]);
                        if (error)
                            return "splits." + error;
                    }
                }
                if (message.endBlockHeight != null && message.hasOwnProperty("endBlockHeight"))
                    if (!$util.isInteger(message.endBlockHeight))
                        return "endBlockHeight: integer expected";
                return null;
            };
    
            /**
             * Creates a SplitContract message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.SplitContract
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.SplitContract} SplitContract
             */
            SplitContract.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.SplitContract)
                    return object;
                var message = new $root.serialization.SplitContract();
                if (object.initiatorAddress != null)
                    if (typeof object.initiatorAddress === "string")
                        $util.base64.decode(object.initiatorAddress, message.initiatorAddress = $util.newBuffer($util.base64.length(object.initiatorAddress)), 0);
                    else if (object.initiatorAddress.length)
                        message.initiatorAddress = object.initiatorAddress;
                if (object.resourceID != null)
                    if (typeof object.resourceID === "string")
                        $util.base64.decode(object.resourceID, message.resourceID = $util.newBuffer($util.base64.length(object.resourceID)), 0);
                    else if (object.resourceID.length)
                        message.resourceID = object.resourceID;
                if (object.splits) {
                    if (!Array.isArray(object.splits))
                        throw TypeError(".serialization.SplitContract.splits: array expected");
                    message.splits = [];
                    for (var i = 0; i < object.splits.length; ++i) {
                        if (typeof object.splits[i] !== "object")
                            throw TypeError(".serialization.SplitContract.splits: object expected");
                        message.splits[i] = $root.serialization.Split.fromObject(object.splits[i]);
                    }
                }
                if (object.endBlockHeight != null)
                    message.endBlockHeight = object.endBlockHeight | 0;
                return message;
            };
    
            /**
             * Creates a plain object from a SplitContract message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.SplitContract
             * @static
             * @param {serialization.SplitContract} message SplitContract
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            SplitContract.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.arrays || options.defaults)
                    object.splits = [];
                if (options.defaults) {
                    object.initiatorAddress = options.bytes === String ? "" : [];
                    object.resourceID = options.bytes === String ? "" : [];
                    object.endBlockHeight = 0;
                }
                if (message.initiatorAddress != null && message.hasOwnProperty("initiatorAddress"))
                    object.initiatorAddress = options.bytes === String ? $util.base64.encode(message.initiatorAddress, 0, message.initiatorAddress.length) : options.bytes === Array ? Array.prototype.slice.call(message.initiatorAddress) : message.initiatorAddress;
                if (message.resourceID != null && message.hasOwnProperty("resourceID"))
                    object.resourceID = options.bytes === String ? $util.base64.encode(message.resourceID, 0, message.resourceID.length) : options.bytes === Array ? Array.prototype.slice.call(message.resourceID) : message.resourceID;
                if (message.splits && message.splits.length) {
                    object.splits = [];
                    for (var j = 0; j < message.splits.length; ++j)
                        object.splits[j] = $root.serialization.Split.toObject(message.splits[j], options);
                }
                if (message.endBlockHeight != null && message.hasOwnProperty("endBlockHeight"))
                    object.endBlockHeight = message.endBlockHeight;
                return object;
            };
    
            /**
             * Converts this SplitContract to JSON.
             * @function toJSON
             * @memberof serialization.SplitContract
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            SplitContract.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return SplitContract;
        })();
    
        serialization.SplitContractTx = (function() {
    
            /**
             * Properties of a SplitContractTx.
             * @memberof serialization
             * @interface ISplitContractTx
             * @property {number|Long|null} [gas] SplitContractTx gas
             * @property {serialization.ICoin|null} [fee] SplitContractTx fee
             * @property {Uint8Array|null} [resourceID] SplitContractTx resourceID
             * @property {serialization.ITxInput|null} [initiator] SplitContractTx initiator
             * @property {Array.<serialization.ISplit>|null} [splits] SplitContractTx splits
             * @property {number|Long|null} [duration] SplitContractTx duration
             */
    
            /**
             * Constructs a new SplitContractTx.
             * @memberof serialization
             * @classdesc Represents a SplitContractTx.
             * @implements ISplitContractTx
             * @constructor
             * @param {serialization.ISplitContractTx=} [properties] Properties to set
             */
            function SplitContractTx(properties) {
                this.splits = [];
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * SplitContractTx gas.
             * @member {number|Long} gas
             * @memberof serialization.SplitContractTx
             * @instance
             */
            SplitContractTx.prototype.gas = $util.Long ? $util.Long.fromBits(0,0,false) : 0;
    
            /**
             * SplitContractTx fee.
             * @member {serialization.ICoin|null|undefined} fee
             * @memberof serialization.SplitContractTx
             * @instance
             */
            SplitContractTx.prototype.fee = null;
    
            /**
             * SplitContractTx resourceID.
             * @member {Uint8Array} resourceID
             * @memberof serialization.SplitContractTx
             * @instance
             */
            SplitContractTx.prototype.resourceID = $util.newBuffer([]);
    
            /**
             * SplitContractTx initiator.
             * @member {serialization.ITxInput|null|undefined} initiator
             * @memberof serialization.SplitContractTx
             * @instance
             */
            SplitContractTx.prototype.initiator = null;
    
            /**
             * SplitContractTx splits.
             * @member {Array.<serialization.ISplit>} splits
             * @memberof serialization.SplitContractTx
             * @instance
             */
            SplitContractTx.prototype.splits = $util.emptyArray;
    
            /**
             * SplitContractTx duration.
             * @member {number|Long} duration
             * @memberof serialization.SplitContractTx
             * @instance
             */
            SplitContractTx.prototype.duration = $util.Long ? $util.Long.fromBits(0,0,false) : 0;
    
            /**
             * Creates a new SplitContractTx instance using the specified properties.
             * @function create
             * @memberof serialization.SplitContractTx
             * @static
             * @param {serialization.ISplitContractTx=} [properties] Properties to set
             * @returns {serialization.SplitContractTx} SplitContractTx instance
             */
            SplitContractTx.create = function create(properties) {
                return new SplitContractTx(properties);
            };
    
            /**
             * Encodes the specified SplitContractTx message. Does not implicitly {@link serialization.SplitContractTx.verify|verify} messages.
             * @function encode
             * @memberof serialization.SplitContractTx
             * @static
             * @param {serialization.ISplitContractTx} message SplitContractTx message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            SplitContractTx.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.gas != null && message.hasOwnProperty("gas"))
                    writer.uint32(/* id 1, wireType 0 =*/8).int64(message.gas);
                if (message.fee != null && message.hasOwnProperty("fee"))
                    $root.serialization.Coin.encode(message.fee, writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
                if (message.resourceID != null && message.hasOwnProperty("resourceID"))
                    writer.uint32(/* id 3, wireType 2 =*/26).bytes(message.resourceID);
                if (message.initiator != null && message.hasOwnProperty("initiator"))
                    $root.serialization.TxInput.encode(message.initiator, writer.uint32(/* id 4, wireType 2 =*/34).fork()).ldelim();
                if (message.splits != null && message.splits.length)
                    for (var i = 0; i < message.splits.length; ++i)
                        $root.serialization.Split.encode(message.splits[i], writer.uint32(/* id 5, wireType 2 =*/42).fork()).ldelim();
                if (message.duration != null && message.hasOwnProperty("duration"))
                    writer.uint32(/* id 6, wireType 0 =*/48).int64(message.duration);
                return writer;
            };
    
            /**
             * Encodes the specified SplitContractTx message, length delimited. Does not implicitly {@link serialization.SplitContractTx.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.SplitContractTx
             * @static
             * @param {serialization.ISplitContractTx} message SplitContractTx message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            SplitContractTx.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes a SplitContractTx message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.SplitContractTx
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.SplitContractTx} SplitContractTx
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            SplitContractTx.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.SplitContractTx();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.gas = reader.int64();
                        break;
                    case 2:
                        message.fee = $root.serialization.Coin.decode(reader, reader.uint32());
                        break;
                    case 3:
                        message.resourceID = reader.bytes();
                        break;
                    case 4:
                        message.initiator = $root.serialization.TxInput.decode(reader, reader.uint32());
                        break;
                    case 5:
                        if (!(message.splits && message.splits.length))
                            message.splits = [];
                        message.splits.push($root.serialization.Split.decode(reader, reader.uint32()));
                        break;
                    case 6:
                        message.duration = reader.int64();
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes a SplitContractTx message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.SplitContractTx
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.SplitContractTx} SplitContractTx
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            SplitContractTx.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies a SplitContractTx message.
             * @function verify
             * @memberof serialization.SplitContractTx
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            SplitContractTx.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.gas != null && message.hasOwnProperty("gas"))
                    if (!$util.isInteger(message.gas) && !(message.gas && $util.isInteger(message.gas.low) && $util.isInteger(message.gas.high)))
                        return "gas: integer|Long expected";
                if (message.fee != null && message.hasOwnProperty("fee")) {
                    var error = $root.serialization.Coin.verify(message.fee);
                    if (error)
                        return "fee." + error;
                }
                if (message.resourceID != null && message.hasOwnProperty("resourceID"))
                    if (!(message.resourceID && typeof message.resourceID.length === "number" || $util.isString(message.resourceID)))
                        return "resourceID: buffer expected";
                if (message.initiator != null && message.hasOwnProperty("initiator")) {
                    var error = $root.serialization.TxInput.verify(message.initiator);
                    if (error)
                        return "initiator." + error;
                }
                if (message.splits != null && message.hasOwnProperty("splits")) {
                    if (!Array.isArray(message.splits))
                        return "splits: array expected";
                    for (var i = 0; i < message.splits.length; ++i) {
                        var error = $root.serialization.Split.verify(message.splits[i]);
                        if (error)
                            return "splits." + error;
                    }
                }
                if (message.duration != null && message.hasOwnProperty("duration"))
                    if (!$util.isInteger(message.duration) && !(message.duration && $util.isInteger(message.duration.low) && $util.isInteger(message.duration.high)))
                        return "duration: integer|Long expected";
                return null;
            };
    
            /**
             * Creates a SplitContractTx message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.SplitContractTx
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.SplitContractTx} SplitContractTx
             */
            SplitContractTx.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.SplitContractTx)
                    return object;
                var message = new $root.serialization.SplitContractTx();
                if (object.gas != null)
                    if ($util.Long)
                        (message.gas = $util.Long.fromValue(object.gas)).unsigned = false;
                    else if (typeof object.gas === "string")
                        message.gas = parseInt(object.gas, 10);
                    else if (typeof object.gas === "number")
                        message.gas = object.gas;
                    else if (typeof object.gas === "object")
                        message.gas = new $util.LongBits(object.gas.low >>> 0, object.gas.high >>> 0).toNumber();
                if (object.fee != null) {
                    if (typeof object.fee !== "object")
                        throw TypeError(".serialization.SplitContractTx.fee: object expected");
                    message.fee = $root.serialization.Coin.fromObject(object.fee);
                }
                if (object.resourceID != null)
                    if (typeof object.resourceID === "string")
                        $util.base64.decode(object.resourceID, message.resourceID = $util.newBuffer($util.base64.length(object.resourceID)), 0);
                    else if (object.resourceID.length)
                        message.resourceID = object.resourceID;
                if (object.initiator != null) {
                    if (typeof object.initiator !== "object")
                        throw TypeError(".serialization.SplitContractTx.initiator: object expected");
                    message.initiator = $root.serialization.TxInput.fromObject(object.initiator);
                }
                if (object.splits) {
                    if (!Array.isArray(object.splits))
                        throw TypeError(".serialization.SplitContractTx.splits: array expected");
                    message.splits = [];
                    for (var i = 0; i < object.splits.length; ++i) {
                        if (typeof object.splits[i] !== "object")
                            throw TypeError(".serialization.SplitContractTx.splits: object expected");
                        message.splits[i] = $root.serialization.Split.fromObject(object.splits[i]);
                    }
                }
                if (object.duration != null)
                    if ($util.Long)
                        (message.duration = $util.Long.fromValue(object.duration)).unsigned = false;
                    else if (typeof object.duration === "string")
                        message.duration = parseInt(object.duration, 10);
                    else if (typeof object.duration === "number")
                        message.duration = object.duration;
                    else if (typeof object.duration === "object")
                        message.duration = new $util.LongBits(object.duration.low >>> 0, object.duration.high >>> 0).toNumber();
                return message;
            };
    
            /**
             * Creates a plain object from a SplitContractTx message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.SplitContractTx
             * @static
             * @param {serialization.SplitContractTx} message SplitContractTx
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            SplitContractTx.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.arrays || options.defaults)
                    object.splits = [];
                if (options.defaults) {
                    if ($util.Long) {
                        var long = new $util.Long(0, 0, false);
                        object.gas = options.longs === String ? long.toString() : options.longs === Number ? long.toNumber() : long;
                    } else
                        object.gas = options.longs === String ? "0" : 0;
                    object.fee = null;
                    object.resourceID = options.bytes === String ? "" : [];
                    object.initiator = null;
                    if ($util.Long) {
                        var long = new $util.Long(0, 0, false);
                        object.duration = options.longs === String ? long.toString() : options.longs === Number ? long.toNumber() : long;
                    } else
                        object.duration = options.longs === String ? "0" : 0;
                }
                if (message.gas != null && message.hasOwnProperty("gas"))
                    if (typeof message.gas === "number")
                        object.gas = options.longs === String ? String(message.gas) : message.gas;
                    else
                        object.gas = options.longs === String ? $util.Long.prototype.toString.call(message.gas) : options.longs === Number ? new $util.LongBits(message.gas.low >>> 0, message.gas.high >>> 0).toNumber() : message.gas;
                if (message.fee != null && message.hasOwnProperty("fee"))
                    object.fee = $root.serialization.Coin.toObject(message.fee, options);
                if (message.resourceID != null && message.hasOwnProperty("resourceID"))
                    object.resourceID = options.bytes === String ? $util.base64.encode(message.resourceID, 0, message.resourceID.length) : options.bytes === Array ? Array.prototype.slice.call(message.resourceID) : message.resourceID;
                if (message.initiator != null && message.hasOwnProperty("initiator"))
                    object.initiator = $root.serialization.TxInput.toObject(message.initiator, options);
                if (message.splits && message.splits.length) {
                    object.splits = [];
                    for (var j = 0; j < message.splits.length; ++j)
                        object.splits[j] = $root.serialization.Split.toObject(message.splits[j], options);
                }
                if (message.duration != null && message.hasOwnProperty("duration"))
                    if (typeof message.duration === "number")
                        object.duration = options.longs === String ? String(message.duration) : message.duration;
                    else
                        object.duration = options.longs === String ? $util.Long.prototype.toString.call(message.duration) : options.longs === Number ? new $util.LongBits(message.duration.low >>> 0, message.duration.high >>> 0).toNumber() : message.duration;
                return object;
            };
    
            /**
             * Converts this SplitContractTx to JSON.
             * @function toJSON
             * @memberof serialization.SplitContractTx
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            SplitContractTx.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return SplitContractTx;
        })();
    
        serialization.UpdateValidatorsTx = (function() {
    
            /**
             * Properties of an UpdateValidatorsTx.
             * @memberof serialization
             * @interface IUpdateValidatorsTx
             * @property {number|Long|null} [gas] UpdateValidatorsTx gas
             * @property {serialization.ICoin|null} [fee] UpdateValidatorsTx fee
             * @property {serialization.ITxInput|null} [proposer] UpdateValidatorsTx proposer
             * @property {Array.<serialization.IValidator>|null} [validators] UpdateValidatorsTx validators
             */
    
            /**
             * Constructs a new UpdateValidatorsTx.
             * @memberof serialization
             * @classdesc Represents an UpdateValidatorsTx.
             * @implements IUpdateValidatorsTx
             * @constructor
             * @param {serialization.IUpdateValidatorsTx=} [properties] Properties to set
             */
            function UpdateValidatorsTx(properties) {
                this.validators = [];
                if (properties)
                    for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                        if (properties[keys[i]] != null)
                            this[keys[i]] = properties[keys[i]];
            }
    
            /**
             * UpdateValidatorsTx gas.
             * @member {number|Long} gas
             * @memberof serialization.UpdateValidatorsTx
             * @instance
             */
            UpdateValidatorsTx.prototype.gas = $util.Long ? $util.Long.fromBits(0,0,false) : 0;
    
            /**
             * UpdateValidatorsTx fee.
             * @member {serialization.ICoin|null|undefined} fee
             * @memberof serialization.UpdateValidatorsTx
             * @instance
             */
            UpdateValidatorsTx.prototype.fee = null;
    
            /**
             * UpdateValidatorsTx proposer.
             * @member {serialization.ITxInput|null|undefined} proposer
             * @memberof serialization.UpdateValidatorsTx
             * @instance
             */
            UpdateValidatorsTx.prototype.proposer = null;
    
            /**
             * UpdateValidatorsTx validators.
             * @member {Array.<serialization.IValidator>} validators
             * @memberof serialization.UpdateValidatorsTx
             * @instance
             */
            UpdateValidatorsTx.prototype.validators = $util.emptyArray;
    
            /**
             * Creates a new UpdateValidatorsTx instance using the specified properties.
             * @function create
             * @memberof serialization.UpdateValidatorsTx
             * @static
             * @param {serialization.IUpdateValidatorsTx=} [properties] Properties to set
             * @returns {serialization.UpdateValidatorsTx} UpdateValidatorsTx instance
             */
            UpdateValidatorsTx.create = function create(properties) {
                return new UpdateValidatorsTx(properties);
            };
    
            /**
             * Encodes the specified UpdateValidatorsTx message. Does not implicitly {@link serialization.UpdateValidatorsTx.verify|verify} messages.
             * @function encode
             * @memberof serialization.UpdateValidatorsTx
             * @static
             * @param {serialization.IUpdateValidatorsTx} message UpdateValidatorsTx message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            UpdateValidatorsTx.encode = function encode(message, writer) {
                if (!writer)
                    writer = $Writer.create();
                if (message.gas != null && message.hasOwnProperty("gas"))
                    writer.uint32(/* id 1, wireType 0 =*/8).int64(message.gas);
                if (message.fee != null && message.hasOwnProperty("fee"))
                    $root.serialization.Coin.encode(message.fee, writer.uint32(/* id 2, wireType 2 =*/18).fork()).ldelim();
                if (message.proposer != null && message.hasOwnProperty("proposer"))
                    $root.serialization.TxInput.encode(message.proposer, writer.uint32(/* id 3, wireType 2 =*/26).fork()).ldelim();
                if (message.validators != null && message.validators.length)
                    for (var i = 0; i < message.validators.length; ++i)
                        $root.serialization.Validator.encode(message.validators[i], writer.uint32(/* id 4, wireType 2 =*/34).fork()).ldelim();
                return writer;
            };
    
            /**
             * Encodes the specified UpdateValidatorsTx message, length delimited. Does not implicitly {@link serialization.UpdateValidatorsTx.verify|verify} messages.
             * @function encodeDelimited
             * @memberof serialization.UpdateValidatorsTx
             * @static
             * @param {serialization.IUpdateValidatorsTx} message UpdateValidatorsTx message or plain object to encode
             * @param {$protobuf.Writer} [writer] Writer to encode to
             * @returns {$protobuf.Writer} Writer
             */
            UpdateValidatorsTx.encodeDelimited = function encodeDelimited(message, writer) {
                return this.encode(message, writer).ldelim();
            };
    
            /**
             * Decodes an UpdateValidatorsTx message from the specified reader or buffer.
             * @function decode
             * @memberof serialization.UpdateValidatorsTx
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @param {number} [length] Message length if known beforehand
             * @returns {serialization.UpdateValidatorsTx} UpdateValidatorsTx
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            UpdateValidatorsTx.decode = function decode(reader, length) {
                if (!(reader instanceof $Reader))
                    reader = $Reader.create(reader);
                var end = length === undefined ? reader.len : reader.pos + length, message = new $root.serialization.UpdateValidatorsTx();
                while (reader.pos < end) {
                    var tag = reader.uint32();
                    switch (tag >>> 3) {
                    case 1:
                        message.gas = reader.int64();
                        break;
                    case 2:
                        message.fee = $root.serialization.Coin.decode(reader, reader.uint32());
                        break;
                    case 3:
                        message.proposer = $root.serialization.TxInput.decode(reader, reader.uint32());
                        break;
                    case 4:
                        if (!(message.validators && message.validators.length))
                            message.validators = [];
                        message.validators.push($root.serialization.Validator.decode(reader, reader.uint32()));
                        break;
                    default:
                        reader.skipType(tag & 7);
                        break;
                    }
                }
                return message;
            };
    
            /**
             * Decodes an UpdateValidatorsTx message from the specified reader or buffer, length delimited.
             * @function decodeDelimited
             * @memberof serialization.UpdateValidatorsTx
             * @static
             * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
             * @returns {serialization.UpdateValidatorsTx} UpdateValidatorsTx
             * @throws {Error} If the payload is not a reader or valid buffer
             * @throws {$protobuf.util.ProtocolError} If required fields are missing
             */
            UpdateValidatorsTx.decodeDelimited = function decodeDelimited(reader) {
                if (!(reader instanceof $Reader))
                    reader = new $Reader(reader);
                return this.decode(reader, reader.uint32());
            };
    
            /**
             * Verifies an UpdateValidatorsTx message.
             * @function verify
             * @memberof serialization.UpdateValidatorsTx
             * @static
             * @param {Object.<string,*>} message Plain object to verify
             * @returns {string|null} `null` if valid, otherwise the reason why it is not
             */
            UpdateValidatorsTx.verify = function verify(message) {
                if (typeof message !== "object" || message === null)
                    return "object expected";
                if (message.gas != null && message.hasOwnProperty("gas"))
                    if (!$util.isInteger(message.gas) && !(message.gas && $util.isInteger(message.gas.low) && $util.isInteger(message.gas.high)))
                        return "gas: integer|Long expected";
                if (message.fee != null && message.hasOwnProperty("fee")) {
                    var error = $root.serialization.Coin.verify(message.fee);
                    if (error)
                        return "fee." + error;
                }
                if (message.proposer != null && message.hasOwnProperty("proposer")) {
                    var error = $root.serialization.TxInput.verify(message.proposer);
                    if (error)
                        return "proposer." + error;
                }
                if (message.validators != null && message.hasOwnProperty("validators")) {
                    if (!Array.isArray(message.validators))
                        return "validators: array expected";
                    for (var i = 0; i < message.validators.length; ++i) {
                        var error = $root.serialization.Validator.verify(message.validators[i]);
                        if (error)
                            return "validators." + error;
                    }
                }
                return null;
            };
    
            /**
             * Creates an UpdateValidatorsTx message from a plain object. Also converts values to their respective internal types.
             * @function fromObject
             * @memberof serialization.UpdateValidatorsTx
             * @static
             * @param {Object.<string,*>} object Plain object
             * @returns {serialization.UpdateValidatorsTx} UpdateValidatorsTx
             */
            UpdateValidatorsTx.fromObject = function fromObject(object) {
                if (object instanceof $root.serialization.UpdateValidatorsTx)
                    return object;
                var message = new $root.serialization.UpdateValidatorsTx();
                if (object.gas != null)
                    if ($util.Long)
                        (message.gas = $util.Long.fromValue(object.gas)).unsigned = false;
                    else if (typeof object.gas === "string")
                        message.gas = parseInt(object.gas, 10);
                    else if (typeof object.gas === "number")
                        message.gas = object.gas;
                    else if (typeof object.gas === "object")
                        message.gas = new $util.LongBits(object.gas.low >>> 0, object.gas.high >>> 0).toNumber();
                if (object.fee != null) {
                    if (typeof object.fee !== "object")
                        throw TypeError(".serialization.UpdateValidatorsTx.fee: object expected");
                    message.fee = $root.serialization.Coin.fromObject(object.fee);
                }
                if (object.proposer != null) {
                    if (typeof object.proposer !== "object")
                        throw TypeError(".serialization.UpdateValidatorsTx.proposer: object expected");
                    message.proposer = $root.serialization.TxInput.fromObject(object.proposer);
                }
                if (object.validators) {
                    if (!Array.isArray(object.validators))
                        throw TypeError(".serialization.UpdateValidatorsTx.validators: array expected");
                    message.validators = [];
                    for (var i = 0; i < object.validators.length; ++i) {
                        if (typeof object.validators[i] !== "object")
                            throw TypeError(".serialization.UpdateValidatorsTx.validators: object expected");
                        message.validators[i] = $root.serialization.Validator.fromObject(object.validators[i]);
                    }
                }
                return message;
            };
    
            /**
             * Creates a plain object from an UpdateValidatorsTx message. Also converts values to other types if specified.
             * @function toObject
             * @memberof serialization.UpdateValidatorsTx
             * @static
             * @param {serialization.UpdateValidatorsTx} message UpdateValidatorsTx
             * @param {$protobuf.IConversionOptions} [options] Conversion options
             * @returns {Object.<string,*>} Plain object
             */
            UpdateValidatorsTx.toObject = function toObject(message, options) {
                if (!options)
                    options = {};
                var object = {};
                if (options.arrays || options.defaults)
                    object.validators = [];
                if (options.defaults) {
                    if ($util.Long) {
                        var long = new $util.Long(0, 0, false);
                        object.gas = options.longs === String ? long.toString() : options.longs === Number ? long.toNumber() : long;
                    } else
                        object.gas = options.longs === String ? "0" : 0;
                    object.fee = null;
                    object.proposer = null;
                }
                if (message.gas != null && message.hasOwnProperty("gas"))
                    if (typeof message.gas === "number")
                        object.gas = options.longs === String ? String(message.gas) : message.gas;
                    else
                        object.gas = options.longs === String ? $util.Long.prototype.toString.call(message.gas) : options.longs === Number ? new $util.LongBits(message.gas.low >>> 0, message.gas.high >>> 0).toNumber() : message.gas;
                if (message.fee != null && message.hasOwnProperty("fee"))
                    object.fee = $root.serialization.Coin.toObject(message.fee, options);
                if (message.proposer != null && message.hasOwnProperty("proposer"))
                    object.proposer = $root.serialization.TxInput.toObject(message.proposer, options);
                if (message.validators && message.validators.length) {
                    object.validators = [];
                    for (var j = 0; j < message.validators.length; ++j)
                        object.validators[j] = $root.serialization.Validator.toObject(message.validators[j], options);
                }
                return object;
            };
    
            /**
             * Converts this UpdateValidatorsTx to JSON.
             * @function toJSON
             * @memberof serialization.UpdateValidatorsTx
             * @instance
             * @returns {Object.<string,*>} JSON object
             */
            UpdateValidatorsTx.prototype.toJSON = function toJSON() {
                return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
            };
    
            return UpdateValidatorsTx;
        })();
    
        return serialization;
    })();

    return $root;
});
